# Chijik Scheduler Plugins
치지직(Chzzk) 스트리밍 서비스 워크로드 최적화를 위한 커스텀 쿠버네티스 스케줄러 플러그인 모음.

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

## 벤치마킹 결과

### 환경
- kind v0.27.0 (control-plane 1 + worker 2)
- k8s v1.29.14
- transcoder 워크로드 Pod 10개 동시 배포

### 노드 BW 설정
| 노드 | BW 사용률 |
|------|---------|
| benchmark-worker | 30% |
| benchmark-worker2 | 80% |

### 결과

**Round 1 - 기본 스케줄러:**
| 노드 | BW 사용률 | 배치된 Pod 수 |
|------|---------|------------|
| benchmark-worker | 30% | 5개 |
| benchmark-worker2 | 80% | 5개 |

→ BW 상관없이 균등 배포. BW 80% 노드에도 동일하게 배치.

**Round 2 - chijik-scheduler:**
| 노드 | BW 사용률 | 배치된 Pod 수 |
|------|---------|------------|
| benchmark-worker | 30% | 10개 |
| benchmark-worker2 | 80% | 0개 |

→ BW 80% 노드 완전 회피. BW 여유있는 노드에만 배치.

### 요약
| 지표 | 기본 스케줄러 | chijik-scheduler |
|------|------------|----------------|
| BW 80% 노드 배치 Pod 수 | 5개 | 0개 |
| BW 30% 노드 배치 Pod 수 | 5개 | 10개 |
| BW 인식 여부 | ❌ | ✅ |

### 한계 및 향후 계획
현재 구현은 노드 어노테이션(`chijik.io/bandwidth-usage`) 기반 프로토타입이다.
실제 BW 사용률은 수동으로 설정해야 하며, 자동 수집되지 않는다.

```
현재:
  kubectl annotate node <node> chijik.io/bandwidth-usage="0.6"  # 수동

향후:
  node_exporter → Prometheus → 컨트롤러가 자동 업데이트
```

## 향후 계획
- [ ] Prometheus 쿼리 기반 실시간 BW 사용률 자동 연동
- [ ] BroadcastController (CRD) 연동 (PredictiveEnqueue 완전 구현)
- [ ] GPU 사용률 기반 Score 플러그인 추가
- [ ] ConnectionScore 플러그인 (WebSocket 동시 연결 수 기반)
