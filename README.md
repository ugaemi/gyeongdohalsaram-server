# gyeongdohalsaram-server

경도할사람 (경찰 vs 도둑) 멀티플레이어 게임 서버

Godot 4.6으로 만든 2D 경찰 vs 도둑 iOS 게임의 온라인 멀티플레이 서버입니다.

## 기술 스택

- Go 1.23+
- WebSocket (gorilla/websocket)

## 빠른 시작

```bash
# 빌드
make build

# 실행
make run

# 테스트
make test
```

## API 엔드포인트

| 엔드포인트 | 설명 |
|-----------|------|
| `GET /health` | 헬스체크 |
| `GET /ws` | WebSocket 연결 |

## 환경변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `PORT` | `8080` | 서버 포트 |
| `LOG_LEVEL` | `info` | 로그 레벨 |
| `LOG_FORMAT` | `text` | 로그 포맷 |

## 라이선스

Private
