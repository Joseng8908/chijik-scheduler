# TranscoderFilter Plugin

## 개요
네트워크 대역폭 사용률이 임계치를 초과한 노드를 필터링하는 Filter 플러그인.
BandwidthScore와 역할 분담:
- TranscoderFilter → 90% 초과 노드 완전 제거
- BandwidthScore   → 남은 노드 중 BW 잔량으로 순위 결정

## 동작 방식
1. `chijik.io/workload=transcoder` 레이블 가진 Pod에만 적용
2. 노드 어노테이션 `chijik.io/bandwidth-usage` 에서 BW 사용률 읽기
3. 임계치(기본 90%) 초과 노드 → Unschedulable 반환 (완전 제거)
4. 미만 노드 → 통과

## 설정
| 항목 | 기본값 | 설명 |
|------|--------|------|
| bwThreshold | 0.9 | BW 사용률 임계치 (0.0~1.0) |

## 사용 예시

### Pod 레이블 설정
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: transcoder
  labels:
    chijik.io/workload: transcoder
```

### 스케줄러 프로파일 설정
```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: chijik-scheduler
    plugins:
      filter:
        enabled:
          - name: TranscoderFilter
```

## 향후 계획
- [ ] Prometheus 쿼리 기반 실시간 BW 사용률 연동
- [ ] bwThreshold 외부 설정 가능하도록 개선
