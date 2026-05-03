# BandwidthScore Plugin

## 개요
노드의 네트워크 대역폭(BW) 잔량을 기반으로 점수를 계산하는 Score 플러그인.
치지직 스트리밍 트랜스코딩 워크로드의 네트워크 핫스팟 노드 회피를 목적으로 설계.

## 동작 방식
1. 노드 어노테이션 `chijik.io/bandwidth-usage` 에서 BW 사용률 읽기
2. 임계치(기본 80%) 초과 노드 → 0점
3. 미만 노드 → `(1 - 사용률) * 100` 점수 부여
4. 점수 높은 노드에 Pod 배치

## 설정
| 항목 | 기본값 | 설명 |
|------|--------|------|
| bwThreshold | 0.8 | BW 사용률 임계치 (0.0~1.0) |

## 사용 예시

### 노드 어노테이션 설정
```bash
kubectl annotate node <node-name> chijik.io/bandwidth-usage="0.6"
```

### 스케줄러 프로파일 설정
```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: chijik-scheduler
    plugins:
      score:
        enabled:
          - name: BandwidthScore
            weight: 10
```

## 향후 계획
- [ ] Prometheus 쿼리 기반 실시간 BW 사용률 연동
- [ ] bwThreshold 외부 설정 가능하도록 개선
