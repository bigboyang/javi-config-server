# javi-config-server 아키텍처

> 이 문서는 PR 마다 자동 생성/갱신됩니다.

## 1. 이게 뭐고 왜 있는가

`javi-config-server`는 javi 시스템의 **중앙 설정 서버**다. APM(Application
Performance Monitoring) 에이전트들이 폴링해서 가져가는 "리모트 설정값"
(샘플링 비율, 계측 on/off, 배치 크기 같은 것들)을 한곳에서 관리하고
배포한다.

이런 서버가 필요한 이유는 단순하다: 각 서비스에 배포된 APM 에이전트의
동작(얼마나 샘플링할지, 긴급 상황에 계측을 끌지, 배치를 얼마나 크게
보낼지 등)을 코드 재배포 없이 실시간으로 바꿀 수 있어야 하기 때문이다.
이 서버는 그 설정값을 들고 있다가, 에이전트가 주기적으로 물어보면
(polling) 현재 값을 내려주는 역할을 한다. 동시에 사람(또는 대시보드)이
그 값을 조회·수정할 수 있는 HTTP API도 제공한다.

## 2. 전체 아키텍처

코드는 의도적으로 아주 단순한 3계층 구조다.

```
main.go              ── 진입점: HTTP 서버 기동 + graceful shutdown
  │
  ├─ internal/config/handler.go  ── HTTP 라우팅 / 요청·응답 변환 (Handler)
  │        │
  │        ▼
  ├─ internal/config/service.go  ── 핵심 로직: 상태 보관, 버전 관리, 영속화 (Service)
  │        │
  │        ▼
  └─ internal/config/model.go    ── 데이터 모델 + 유효성 검사 (RemoteConfig, ServiceKey)
```

- **main.go**: `Service`와 `Handler`를 만들어 연결하고, `:18888` 포트에서
  HTTP 서버를 띄운다. `SIGINT`/`SIGTERM`을 받으면 10초 타임아웃으로
  graceful shutdown을 수행한다.
- **Handler** (`internal/config/handler.go`): 들어오는 HTTP 요청을
  파싱해서 `Service`의 메서드를 호출하고, 결과를 JSON으로 직렬화해
  돌려준다. 비즈니스 로직은 거의 없고 순수하게 라우팅/변환 계층이다.
- **Service** (`internal/config/service.go`): 실제 상태를 들고 있는
  곳. 전역(global) 설정 하나와, 서비스별/환경별 오버라이드(override)
  맵을 메모리에 들고 있으며, 변경이 생길 때마다 버전을 올리고 디스크에
  저장한다.
- **Model** (`internal/config/model.go`): `RemoteConfig` 구조체(필드
  목록)와 그 값들의 유효 범위를 검사하는 `Validate()`, 그리고
  서비스+환경 쌍을 표현하는 `ServiceKey`를 정의한다.

## 3. 데이터/요청 흐름

이 서버에는 사실상 두 종류의 클라이언트가 있고, 흐름이 약간 다르다.

### (A) APM 에이전트의 폴링 흐름 (읽기, 캐시 친화적)

```
APM Agent                         config-server
   │  GET /api/config/remote?service=payment&env=prod
   │  If-None-Match: "<직전 ETag>"
   ├──────────────────────────────────────────────────▶
   │                                  Handler.getForAgent
   │                                    │
   │                                    ▼
   │                          ETag(현재 버전) == If-None-Match ?
   │                                    │
   │                ┌───────────────────┴───────────────────┐
   │              같으면                                  다르면
   │           304 Not Modified                  Service.GetEffective(key)
   │                                                  │  service:env override
   │                                                  │  → service override
   │                                                  │  → global config
   │                                                  ▼
   │                                     200 OK + RemoteConfig + ETag
   ◀──────────────────────────────────────────────────┤
```

핵심은 **버전 = ETag**라는 점이다. `Service.version`(전역 단일
카운터)이 바뀔 때마다 ETag도 바뀌므로, 에이전트는 마지막으로 받은
ETag를 `If-None-Match`로 보내서 변경이 없으면 304만 받고 끝낼 수
있다. 매 폴링마다 전체 설정 JSON을 주고받지 않아도 되게 하려는
설계다.

설정 조회 우선순위(`GetEffective`)는 다음과 같다.

1. `service:env` 정확히 일치하는 오버라이드
2. (env가 있을 때) `service`만 일치하는 오버라이드
3. 둘 다 없으면 전역(global) 설정

### (B) 대시보드/운영자의 관리 흐름 (쓰기)

```
운영자/대시보드 → PUT/PATCH /api/config          → 전역 설정 교체/부분 수정
                → GET /api/config                → 전역 설정 + 버전 + ETag 조회
                → GET  /api/config/services       → 모든 오버라이드 목록
                → GET/PUT/DELETE /api/config/service/{service}?env=...
                                                  → 서비스(+환경)별 오버라이드 관리
```

쓰기 요청은 모두 `Service`를 거쳐 다음 순서로 처리된다.

1. (PUT/PATCH인 경우) `RemoteConfig.Validate()`로 값 범위 검증
2. 통과하면 메모리 상태 갱신 + 버전 atomic 증가
3. `saveState()`로 JSON 파일에 즉시 동기 저장 (영속화)

검증 실패 시 `422 Unprocessable Entity`, JSON 파싱 실패 시
`400 Bad Request`를 돌려준다.

## 4. 디렉터리/모듈 책임

| 경로 | 책임 | 왜 이렇게 나눴나 |
|---|---|---|
| `main.go` | 프로세스 생명주기 (기동, 시그널 처리, graceful shutdown) | 서버 와이어링과 비즈니스 로직을 분리해두면 테스트하기 쉬움 |
| `internal/config/handler.go` | HTTP 라우팅, 요청 파싱, JSON 응답 작성 | 전송 계층(HTTP)을 도메인 로직(Service)과 분리. `Service`는 HTTP를 전혀 모름 |
| `internal/config/service.go` | 상태 보관(전역+오버라이드), 버전/ETag 관리, 동시성 제어, 파일 영속화 | 이 파일이 사실상 이 프로젝트의 "엔진". `sync.Map`과 `atomic.Pointer`로 락 없이 동시 접근을 다룸 |
| `internal/config/model.go` | `RemoteConfig` 필드 정의, 기본값(`defaultConfig`), 유효성 검사 | 데이터 형태와 제약 조건을 한곳에 모아 Handler/Service 양쪽에서 재사용 |
| `run.sh` | 빌드 후 즉시 실행하는 로컬 편의 스크립트 | `go build && exec` 패턴으로 매번 명령어 두 개 칠 필요 없게 |
| `.github/` | PR마다 CLAUDE.md / 이 문서(ARCHITECTURE.md)를 자동 갱신하는 워크플로 | 문서가 코드와 같이 늙지(stale) 않도록 자동화 |

`internal/` 패키지로 둔 이유는 Go 컨벤션상 외부 모듈에서 import하지
못하게 막아, 이 패키지가 이 서버 전용 구현임을 명시하기 위함으로
보인다(레포에 별도 설명은 없음 — 추정).

## 5. 로컬 개발 시작하기

필요한 것: Go 1.22 이상 (go.mod에 명시됨).

```bash
# 가장 빠른 방법
go run .

# 또는 빌드 후 실행 (run.sh가 이걸 해줌)
./run.sh

# 빌드만
go build -o config-server .

# 테스트
go test ./...
```

서버는 기본적으로 `:18888`에서 뜬다(포트는 코드에 하드코딩되어 있고
환경변수로 바꿀 수 없음 — `main.go` 참고).

상태 파일 경로는 `STATE_FILE` 환경변수로 지정 가능하며, 지정하지
않으면 현재 디렉터리의 `config-state.json`을 사용한다. 이 파일이
이미 있으면 그 안의 `global`/`overrides`/`version`을 그대로 불러와
이전 상태를 복원한다. 없으면 `defaultConfig()`의 값으로 새로
시작한다.

빠른 동작 확인:

```bash
curl http://localhost:18888/api/config/remote
curl http://localhost:18888/api/config/remote?service=payment&env=prod
curl -X PATCH "http://localhost:18888/api/config/emergencyOff?value=true"
```

(더 많은 예시는 `internal/config/handler.go` 상단 주석 참고.)

## 6. 알아두면 좋은 함정·트레이드오프

- **버전이 전역 단일 카운터다.** 전역 설정을 바꾸든, 특정 서비스
  오버라이드 하나만 바꾸든 같은 `version`(따라서 같은 ETag)이
  올라간다. 즉 어떤 에이전트든 *아무* 설정이 바뀌면 캐시가 무효화돼
  304 대신 200을 받게 된다 — 변경이 자신과 무관해도 마찬가지다. 폴링
  빈도가 매우 높고 오버라이드가 자주 바뀌는 환경이면 이 점이 캐시
  효율을 깎아먹을 수 있다.
- **저장은 매 쓰기마다 동기적으로, 전체 상태를 덮어쓴다.**
  `saveState()`는 변경 한 번마다 전역+모든 오버라이드를 JSON으로
  통째로 다시 쓴다. 오버라이드가 많아지거나 쓰기가 잦아지면 디스크
  I/O가 병목이 될 수 있다. 트랜잭션이나 원자적 파일 교체(rename)
  로직은 없으므로, 쓰는 도중 프로세스가 죽으면 상태 파일이 깨질
  가능성이 이론상 있다(코드에서 명시적으로 다루지 않음 — 추정).
  파일 쓰기 실패는 로그만 남기고 무시하므로, 디스크가 꽉 찼거나
  쓰기 권한이 없으면 변경사항이 메모리에는 반영됐지만 재시작 시
  사라질 수 있다.
- **인증/인가가 전혀 없다.** 모든 엔드포인트가 누구나 호출 가능한
  평문 HTTP다. 같은 네트워크에 있는 누구든 `PUT /api/config`로 전역
  설정을 바꾸거나 `emergencyOff`를 끌 수 있다. 운영 환경에 노출할
  때는 앞단에 인증/네트워크 격리가 반드시 필요하다(이 레포 안에는
  그런 장치가 없음).
- **그레이스풀 셧다운 타임아웃은 10초로 고정**(`main.go`). 진행 중인
  요청이 이보다 오래 걸리면 강제 종료된다.
- **포트(`:18888`)가 하드코딩**돼 있어 같은 호스트에 여러 인스턴스를
  띄우려면 코드를 고치거나 컨테이너/네임스페이스로 격리해야 한다.
- **CLAUDE.md vs 이 문서의 역할 분리**: `CLAUDE.md`는 AI 에이전트가
  작업할 때 보는 간결한 참조 문서이고, 매 PR마다
  `.github/workflows/sync-claudemd.yml`이 자동으로 갱신을 시도한다.
  이 문서(`docs/ARCHITECTURE.md`)는 그 워크플로의 세 번째 단계로,
  사람 개발자의 온보딩을 돕기 위해 별도로 생성/갱신된다. 두 문서
  모두 자동화 스크립트가 건드리므로, 수동으로 정성 들여 쓴 내용도
  다음 PR에서 AI가 다시 갱신할 수 있다는 점을 기억해두면 좋다.
