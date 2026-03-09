package com.javi.configserver.service;

import com.javi.configserver.model.RemoteConfigDto;
import org.springframework.stereotype.Service;

import java.lang.reflect.Field;
import java.util.concurrent.atomic.AtomicReference;

/**
 * APM Agent Remote Config를 in-memory로 관리한다.
 *
 * AtomicReference로 스레드 안전한 전체 교체를 보장한다.
 * APM Dashboard에서 PUT/PATCH로 설정을 변경하고,
 * Agent는 GET /api/config/remote로 polling한다.
 */
@Service
public class ConfigService {

    private final AtomicReference<RemoteConfigDto> config =
            new AtomicReference<>(new RemoteConfigDto());

    public RemoteConfigDto get() {
        return config.get();
    }

    public void replace(RemoteConfigDto newConfig) {
        config.set(newConfig);
    }

    /**
     * 단일 필드를 업데이트한다.
     * Reflection으로 필드 이름과 매핑하여 타입 변환 후 적용한다.
     *
     * @throws IllegalArgumentException 알 수 없는 필드명이거나 타입 변환 실패 시
     */
    public void patch(String key, String value) {
        RemoteConfigDto current = config.get();
        RemoteConfigDto updated = copy(current);
        applyField(updated, key, value);
        config.set(updated);
    }

    private RemoteConfigDto copy(RemoteConfigDto src) {
        RemoteConfigDto dst = new RemoteConfigDto();
        dst.setHeadSampleRate(src.getHeadSampleRate());
        dst.setTailPolicy(src.getTailPolicy());
        dst.setTargetTps(src.getTargetTps());
        dst.setServiceWeight(src.getServiceWeight());
        dst.setLogInjection(src.isLogInjection());
        dst.setMetrics(src.getMetrics());
        dst.setSpanDrop(src.getSpanDrop());
        dst.setCustomHeaders(src.getCustomHeaders());
        dst.setEmergencyOff(src.isEmergencyOff());
        dst.setServiceDisable(src.getServiceDisable());
        dst.setDropOnFull(src.isDropOnFull());
        dst.setBatchSize(src.getBatchSize());
        dst.setExportDelay(src.getExportDelay());
        dst.setRetryCount(src.getRetryCount());
        return dst;
    }

    private void applyField(RemoteConfigDto dto, String key, String value) {
        try {
            Field field = RemoteConfigDto.class.getDeclaredField(key);
            field.setAccessible(true);
            Class<?> type = field.getType();
            if (type == double.class) {
                field.set(dto, Double.parseDouble(value));
            } else if (type == long.class) {
                field.set(dto, Long.parseLong(value));
            } else if (type == int.class) {
                field.set(dto, Integer.parseInt(value));
            } else if (type == boolean.class) {
                field.set(dto, Boolean.parseBoolean(value));
            } else {
                field.set(dto, value);
            }
        } catch (NoSuchFieldException e) {
            throw new IllegalArgumentException("Unknown config key: " + key);
        } catch (NumberFormatException e) {
            throw new IllegalArgumentException("Invalid value for key '" + key + "': " + value);
        } catch (IllegalAccessException e) {
            throw new IllegalArgumentException("Cannot update field: " + key);
        }
    }
}
