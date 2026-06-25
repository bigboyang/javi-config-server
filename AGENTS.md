<!-- AUTO-MIRROR of CLAUDE.md — 직접 수정 금지 (codex/AGENTS.md) -->

# javi-config-server

javi 시스템의 **설정 서버**. Go로 작성된 HTTP 서버로, 각 컴포넌트에 구성값을 제공한다.

## 아키텍처

- `main.go` — 진입점 (HTTP 서버 기동, graceful shutdown)
- `internal/config` — 설정 로직
- `run.sh` — 실행 스크립트
- module: `github.com/javi/config-server`

## 실행

- 실행: `./run.sh` 또는 `go run .`
- 빌드: `go build -o config-server .`
- 테스트: `go test ./...`

## 규칙·관례

> 코딩 컨벤션·주의사항을 여기에 적어두세요. PR로 코드가 바뀌면 이 영역은 GitHub Actions(Claude)가 자동 보강합니다.

- `AGENTS.md`(codex용)는 이 파일의 미러본이다. CI가 자동으로 동기화하므로 직접 수정하지 말 것.

<!-- AUTO-GENERATED:start (스크립트가 관리. 직접 수정 금지) -->

_아래 구간은 스크립트가 자동 생성합니다. 직접 수정하지 마세요._

### 기술 스택
- Go (`go.mod`)

### 명령어

### 최상위 디렉터리 구조
```
.github
internal
```

<!-- AUTO-GENERATED:end -->
