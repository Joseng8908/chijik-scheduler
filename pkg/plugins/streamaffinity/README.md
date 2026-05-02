# StreamAffinity Plugin

## 개요
같은 스트림의 Ingest Pod와 Transcoder Pod를 같은 노드에 배치하는 Score 플러그인.
같은 노드 배치 시 노드 간 네트워크 통신이 loopback으로 대체되어 레이턴시 및 대역폭 소모 제거.

## 동작 방식
1. `chijik.io/workload=transcoder` 레이블 가진 Pod에만 적용
2. Pod의 `chijik.io/stream-id` 레이블로 스트림 식별
3. 각 노드 순회 → 같은 stream-id의 Ingest Pod 있는지 확인
4. 있으면 → 100점 / 없으면 → 0점

```
[Ingest Pod]  ←loopback→  [Transcoder Pod]
     같은 노드에 배치되면 네트워크 오버헤드 없음
```

## 설정
별도 설정 없음. Pod 레이블로 동작.

## 사용 예시

### Pod 레이블 설정
```yaml
# Ingest Pod
apiVersion: v1
kind: Pod
metadata:
  name: ingest-streamer-a
  labels:
    chijik.io/workload: ingest
    chijik.io/stream-id: streamer-a

---
# Transcoder Pod
apiVersion: v1
kind: Pod
metadata:
  name: transcoder-streamer-a
  labels:
    chijik.io/workload: transcoder
    chijik.io/stream-id: streamer-a
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
          - name: StreamAffinity
            weight: 10
```

## 향후 계획
- [ ] 멀티 노드 환경에서 rack-level affinity 확장
- [ ] Ingest Pod 없을 때 fallback 전략 추가
