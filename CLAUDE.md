# gyeongdohalsaram-server

2D 경찰 vs 도둑 멀티플레이어 게임 서버 (Godot 4.6 클라이언트용)

## 기술 스택

- **언어**: Go 1.23+
- **WebSocket**: `github.com/gorilla/websocket`
- **UUID**: `github.com/google/uuid`
- **테스트**: `github.com/stretchr/testify`
- **로깅**: `log/slog` (stdlib)
- **DB**: `github.com/jackc/pgx/v5` (PostgreSQL)

## 프로젝트 구조

```
cmd/server/main.go          # 엔트리포인트
internal/
  config/config.go           # 환경변수 설정
  ws/                        # WebSocket Hub/Client
  room/                      # 방 관리
  game/                      # 게임 로직, 상수, 플레이어
  handler/                   # 메시지 라우팅 및 핸들러
  account/                   # 계정 모델
  auth/                      # 인증 (Game Center 서명 검증)
  store/                     # DB 저장소 (PostgreSQL)
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
- 게임시간: 180초
- 플레이어: 2-8명 (경찰 최대 2명)

## 환경변수

- `PORT`: 서버 포트 (기본: 8080)
- `LOG_LEVEL`: debug, info, warn, error (기본: info)
- `LOG_FORMAT`: text, json (기본: text)
- `DATABASE_URL`: PostgreSQL 연결 URL (기본: `postgres://postgres:postgres@localhost:5432/gyeongdohalsaram?sslmode=disable`)
- `GC_BUNDLE_IDS`: 허용 Game Center 번들 ID, 콤마 구분 (미설정 시 모든 번들 허용)
- `GC_TIMESTAMP_TOLERANCE`: Game Center 타임스탬프 허용 오차 초 (기본: 300)

## 워크플로우

- 새 기능 계획 시 반드시 GitHub issue로 등록한 뒤 구현 시작
- 모든 issue는 `gyeongdohalsaram mvp` GitHub Project에 연결
- 커밋 메시지에 관련 issue 번호 포함: `feat: 기능 설명 (#이슈번호)`
- 하나의 issue에 대한 작업은 하나의 브랜치에서 진행

## 코드 컨벤션

- 공개 함수에 GoDoc 주석 작성
- 에러는 즉시 반환 (early return)
- 동시성: sync.RWMutex로 공유 상태 보호
- 테스트: testify 사용, 테이블 기반 테스트 선호
