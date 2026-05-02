# Chijik Scheduler Plugins

치지직(Chizzk) 스트리밍 서비스 워크로드 최적화를 위한 커스텀 쿠버네티스 스케줄러 플러그인 모음.

## 배경

기본 쿠버네티스 스케줄러는 CPU/메모리 기반으로만 노드를 선택함.
스트리밍 서비스는 네트워크 대역폭 집중, 낮은 레이턴시, Pod 간 친화성 등 특수한 요구사항이 있음.

```
[스트리머] → RTMP → [Ingest Pod] → [Transcoder Pod] → [CDN] → [시청자]
                                        ↑
                              CPU/GPU 집약 + 네트워크 대역폭 집중
                              → 기본 스케줄러는 이걸 모름
```

## 플러그인 목록

| 플러그인 | Extension Point | 역할 |
|---------|----------------|------|
| [BandwidthScore](./bandwidthscore/) | Score | 노드 네트워크 BW 잔량 기반 점수 계산 |
| [TranscoderFilter](./transcoderfilter/) | Filter | BW 임계치 초과 노드 완전 제거 |
| [StreamAffinity](./streamaffinity/) | Score | Ingest↔Transcoder 같은 노드 친화 |
| [PredictiveEnqueue](./predictiveenqueue/) | PreEnqueue | 방송 예약 기반 미리 리소스 예약 |

## 플러그인 역할 분담

```
Pod 스케줄링 사이클:

PreEnqueue
  └── PredictiveEnqueue  → 방송 시작 5분 전부터 큐 진입 허용

Filter
  └── TranscoderFilter   → BW 90% 초과 노드 제거

Score
  ├── BandwidthScore     → BW 잔량 기반 점수 (가중치 10)
  └── StreamAffinity     → Ingest Pod 있는 노드 우선 (가중치 10)
```

## Pod 레이블/어노테이션 규칙

| 키 | 종류 | 값 예시 | 설명 |
|----|------|---------|------|
| `chijik.io/workload` | label | `ingest`, `transcoder` | 워크로드 타입 |
| `chijik.io/stream-id` | label | `streamer-a` | 스트림 식별자 |
| `chijik.io/bandwidth-usage` | annotation (node) | `0.6` | 노드 BW 사용률 |
| `chijik.io/broadcast-time` | annotation (pod) | `2024-01-01T15:00:00Z` | 방송 예정 시간 |

## 스케줄러 프로파일 전체 설정

```yaml
apiVersion: kubescheduler.config.k8s.io/v1
kind: KubeSchedulerConfiguration
profiles:
  - schedulerName: chijik-scheduler
    plugins:
      preEnqueue:
        enabled:
          - name: PredictiveEnqueue
      filter:
        enabled:
          - name: TranscoderFilter
      score:
        enabled:
          - name: BandwidthScore
            weight: 10
          - name: StreamAffinity
            weight: 10
```

## 향후 계획

- [ ] Prometheus 쿼리 기반 실시간 BW 사용률 연동 (BandwidthScore, TranscoderFilter)
- [ ] BroadcastController (CRD) 연동 (PredictiveEnqueue)
- [ ] GPU 사용률 기반 Score 플러그인 추가
- [ ] 벤치마킹 결과 추가
