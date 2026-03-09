package com.javi.configserver.controller;

import com.javi.configserver.model.RemoteConfigDto;
import com.javi.configserver.service.ConfigService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.Map;

/**
 * APM Agent Remote Config API.
 *
 * <p>Agent polling endpoint:
 * <pre>
 *   GET /api/config/remote   — flat JSON (RemoteConfigPoller.parse() 호환)
 * </pre>
 *
 * <p>APM Dashboard management endpoints:
 * <pre>
 *   GET    /api/config              — 현재 설정 조회
 *   PUT    /api/config              — 전체 설정 교체
 *   PATCH  /api/config/{key}        — 단일 필드 업데이트 (?value=...)
 * </pre>
 *
 * <p>예시:
 * <pre>
 *   curl http://localhost:8888/api/config/remote
 *   curl -X PATCH "http://localhost:8888/api/config/emergencyOff?value=true"
 *   curl -X PATCH "http://localhost:8888/api/config/headSampleRate?value=0.5"
 *   curl -X PUT http://localhost:8888/api/config \
 *        -H "Content-Type: application/json" \
 *        -d '{"headSampleRate":0.5,"emergencyOff":false,"logInjection":true,...}'
 * </pre>
 */
@RestController
@RequestMapping("/api/config")
public class ConfigController {

    private final ConfigService configService;

    public ConfigController(ConfigService configService) {
        this.configService = configService;
    }

    /** Agent polling endpoint — flat JSON 반환 */
    @GetMapping("/remote")
    public RemoteConfigDto getForAgent() {
        return configService.get();
    }

    /** Dashboard: 현재 설정 조회 */
    @GetMapping
    public RemoteConfigDto get() {
        return configService.get();
    }

    /** Dashboard: 전체 설정 교체 */
    @PutMapping
    public ResponseEntity<Map<String, String>> put(@RequestBody RemoteConfigDto newConfig) {
        configService.replace(newConfig);
        return ResponseEntity.ok(Map.of("status", "updated"));
    }

    /** Dashboard: 단일 필드 업데이트 */
    @PatchMapping("/{key}")
    public ResponseEntity<?> patch(@PathVariable String key,
                                   @RequestParam String value) {
        try {
            configService.patch(key, value);
            return ResponseEntity.ok(Map.of("status", "patched", "key", key, "value", value));
        } catch (IllegalArgumentException e) {
            return ResponseEntity.badRequest().body(Map.of("error", e.getMessage()));
        }
    }
}
