package com.javi.configserver.model;

/**
 * APM Agent가 polling하는 RemoteConfig 데이터 모델.
 *
 * Agent의 RemoteConfigPoller.parse()가 기대하는 flat JSON 필드와 1:1 매핑.
 * Jackson이 직렬화 시 flat JSON으로 변환한다.
 */
public class RemoteConfigDto {

    // Sampling
    private double headSampleRate = 1.0;
    private String tailPolicy = "error,slow,cluster";
    private long targetTps = 0;
    private String serviceWeight = "";

    // Instrumentation
    private boolean logInjection = true;
    private String metrics = "all";
    private String spanDrop = "";
    private String customHeaders = "";

    // Emergency
    private boolean emergencyOff = false;
    private String serviceDisable = "";

    // Performance
    private boolean dropOnFull = true;
    private int batchSize = 512;
    private long exportDelay = 5000;
    private int retryCount = 3;

    public double getHeadSampleRate() { return headSampleRate; }
    public void setHeadSampleRate(double headSampleRate) { this.headSampleRate = headSampleRate; }

    public String getTailPolicy() { return tailPolicy; }
    public void setTailPolicy(String tailPolicy) { this.tailPolicy = tailPolicy; }

    public long getTargetTps() { return targetTps; }
    public void setTargetTps(long targetTps) { this.targetTps = targetTps; }

    public String getServiceWeight() { return serviceWeight; }
    public void setServiceWeight(String serviceWeight) { this.serviceWeight = serviceWeight; }

    public boolean isLogInjection() { return logInjection; }
    public void setLogInjection(boolean logInjection) { this.logInjection = logInjection; }

    public String getMetrics() { return metrics; }
    public void setMetrics(String metrics) { this.metrics = metrics; }

    public String getSpanDrop() { return spanDrop; }
    public void setSpanDrop(String spanDrop) { this.spanDrop = spanDrop; }

    public String getCustomHeaders() { return customHeaders; }
    public void setCustomHeaders(String customHeaders) { this.customHeaders = customHeaders; }

    public boolean isEmergencyOff() { return emergencyOff; }
    public void setEmergencyOff(boolean emergencyOff) { this.emergencyOff = emergencyOff; }

    public String getServiceDisable() { return serviceDisable; }
    public void setServiceDisable(String serviceDisable) { this.serviceDisable = serviceDisable; }

    public boolean isDropOnFull() { return dropOnFull; }
    public void setDropOnFull(boolean dropOnFull) { this.dropOnFull = dropOnFull; }

    public int getBatchSize() { return batchSize; }
    public void setBatchSize(int batchSize) { this.batchSize = batchSize; }

    public long getExportDelay() { return exportDelay; }
    public void setExportDelay(long exportDelay) { this.exportDelay = exportDelay; }

    public int getRetryCount() { return retryCount; }
    public void setRetryCount(int retryCount) { this.retryCount = retryCount; }
}
