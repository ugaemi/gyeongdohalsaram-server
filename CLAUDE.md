# gyeongdohalsaram-server

2D 경찰 vs 도둑 멀티플레이어 게임 서버 (Godot 4.6 클라이언트용)

## 기술 스택

- **언어**: Go 1.23+
- **WebSocket**: `github.com/gorilla/websocket`
- **UUID**: `github.com/google/uuid`
- **테스트**: `github.com/stretchr/testify`
- **로깅**: `log/slog` (stdlib)

## 프로젝트 구조

```
cmd/server/main.go          # 엔트리포인트
internal/
  config/config.go           # 환경변수 설정
  ws/                        # WebSocket Hub/Client
  room/                      # 방 관리
  game/                      # 게임 로직, 상수, 플레이어
  handler/                   # 메시지 라우팅 및 핸들러
```

## 빌드 & 실행

```bash
make build    # 바이너리 빌드
make run      # 서버 실행
make test     # 테스트 실행
make lint     # 린트 실행
```

## 아키텍처

- **Hub-Client 패턴**: gorilla/websocket 표준 패턴
- **방별 독립 게임루프**: goroutine (20 TPS)
- **JSON WebSocket 프로토콜**: `{"type": "...", "data": {...}}`

## 게임 상수 (Godot 클라이언트 일치)

- 맵: 3240x5760px
- 이동속도: 400px/s
- 체포시간: 1.5초 (누적)
- 탈옥시간: 2.0초 (연속)
- 게임시간: 120초
- 플레이어: 2-8명 (경찰 최대 2명)

## 환경변수

- `PORT`: 서버 포트 (기본: 8080)
- `LOG_LEVEL`: debug, info, warn, error (기본: info)
- `LOG_FORMAT`: text, json (기본: text)

## 코드 컨벤션

- 공개 함수에 GoDoc 주석 작성
- 에러는 즉시 반환 (early return)
- 동시성: sync.RWMutex로 공유 상태 보호
- 테스트: testify 사용, 테이블 기반 테스트 선호
