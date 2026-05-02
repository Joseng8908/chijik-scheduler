# PredictiveEnqueue Plugin

## 개요
방송 예약 시간 기반으로 스케줄링 큐 진입 시점을 제어하는 PreEnqueue 플러그인.
방송 시작 5분 전부터 Pod를 큐에 올려 리소스를 미리 확보.

## 동작 방식
1. Pod 어노테이션 `chijik.io/broadcast-time` 확인
2. 어노테이션 없으면 → 일반 Pod처럼 바로 큐 진입
3. 현재 시간이 (방송 시작 - 5분) 이전 → 큐 진입 차단
4. 현재 시간이 (방송 시작 - 5분) 이후 → 큐 진입 허용 → 스케줄러가 노드 배치 시작

```
방송 예정: 15:00
큐 진입 허용: 14:55 (5분 전)
         ↓
스케줄러가 노드 선택 → 리소스 확보 완료
         ↓
15:00 방송 시작 → 즉시 트랜스코딩 가능
```

## 설정
| 항목 | 기본값 | 설명 |
|------|--------|------|
| PrewarmDuration | 5분 | 방송 시작 몇 분 전부터 큐에 올릴지 |

## 사용 예시

### Pod 어노테이션 설정
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: transcoder-streamer-a
  annotations:
    chijik.io/broadcast-time: "2024-01-01T15:00:00Z"  # RFC3339 형식
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
      preEnqueue:
        enabled:
          - name: PredictiveEnqueue
```

## 한계 및 향후 계획
- [ ] Pod가 미리 생성돼 있어야 동작함 → BroadcastController (CRD) 연동 필요
- [ ] PrewarmDuration 외부 설정 가능하도록 개선
- [ ] 방송 취소 시 Pod 자동 삭제 로직 필요
