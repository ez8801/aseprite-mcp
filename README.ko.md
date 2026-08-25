[English](README.md) | 한국어

# Aseprite MCP

AI 어시스턴트가 [Aseprite](https://www.aseprite.org/) 스프라이트를 생성하고
편집하고 내보낼 수 있게 해주는 [MCP](https://modelcontextprotocol.io) 서버.

Aseprite의 배치 모드 CLI를 구동하므로 로컬에 설치된 Aseprite가 필요하지만
(1.3.18.2에서 테스트) 플러그인이나 실행 중인 GUI는 필요 없다.

## 도구

### 파일과 문서

| 도구 | 용도 |
| --- | --- |
| `aseprite_health` | 사용 중인 Aseprite 실행 파일과 버전을 보고한다. |
| `create_sprite` | 빈 스프라이트를 만들어 디스크에 쓴다. |
| `get_sprite_info` | 크기, 컬러 모드, 레이어, 프레임, 태그, 팔레트를 보고한다. |
| `save_sprite_as` | 다른 위치나 포맷으로 사본을 저장한다. |

### 그리기

| 도구 | 용도 |
| --- | --- |
| `draw_pixels` | 픽셀마다 각각의 색으로 개별 설정한다. |
| `draw_shapes` | 선, 사각형, 타원을 채우거나 외곽선으로 그린다. |
| `fill_area` | 한 점에서 시작해 플러드 필한다. |
| `clear_area` | 사각 영역이나 셀 전체를 지운다. |
| `stamp_sprites` | 다른 스프라이트에 스프라이트 전체를 복사해 장면을 조립한다. |

### 레이어, 프레임, 태그

| 도구 | 용도 |
| --- | --- |
| `add_layer` | 이미지 레이어 또는 레이어 그룹을 추가한다. |
| `update_layer` | 이름을 바꾸거나 불투명도, 표시 여부, 블렌드 모드를 변경한다. |
| `delete_layer` | 레이어를 삭제한다. 그룹이면 내용까지 함께 삭제된다. |
| `add_frames` | 프레임을 뒤에 붙이거나 중간에 삽입한다. 복사 또는 빈 프레임. |
| `delete_frames` | 번호로 프레임을 삭제한다. |
| `set_frame_durations` | 개별 프레임의 지속 시간을 조정한다. |
| `set_tag` | 애니메이션 태그를 만들거나 교체한다. |
| `delete_tag` | 애니메이션 태그를 삭제한다. |

### 팔레트와 내보내기

| 도구 | 용도 |
| --- | --- |
| `get_palette` | 팔레트 전체를 잘림 없이 읽는다. |
| `set_palette` | 색 목록이나 팔레트 파일로 팔레트를 교체한다. |
| `save_palette` | 팔레트를 별도 파일로 쓴다 (`.gpl`, `.pal`, ...). |
| `resize_sprite` | 지정한 크기나 배율로 크기를 바꾼다. |
| `export_spritesheet` | 모든 프레임을 한 장의 시트로 내보낸다. JSON 데이터 파일은 선택. |

## 프롬프트

서버는 MCP 프롬프트 `animated_character` 하나를 함께 제공한다. 이 도구들로
캐릭터와 애니메이션 상태를 그리는 작업 흐름을 정리해준다: 팔레트와 실루엣,
기본 포즈, 눈으로 확인하는 체크포인트, 상태별 프레임, 태그와 타이밍, 그리고
내보내기. 요청한 상태들로부터 프레임 범위 번호를 자동으로 매긴다.

Claude Code에서는 슬래시 명령으로 나타난다:

```
/aseprite:animated_character
```

| 인자 | 의미 |
| --- | --- |
| `name` | 필수. 파일 이름에 쓰이는 캐릭터 슬러그. |
| `description` | 필수. 캐릭터의 생김새. |
| `outputDir` | 결과를 쓸 절대 경로 디렉터리. |
| `animations` | 쉼표로 구분한 `state:frames` 쌍. 기본값 `idle:6,attack:8`. |
| `size` | 캔버스 크기 `WIDTHxHEIGHT`. 기본값 `64x64`. |

## 호출 동작 방식

호출은 상태를 갖지 않는다. 호출 사이에 열린 채로 유지되는 문서가 없으므로,
모든 도구가 절대 경로를 받아 파일을 직접 읽고 쓴다.

편집 도구는 받은 스프라이트를 제자리에서 다시 쓴다. 호출 한 번에 파일을 한 번
열고 한 번 저장하므로, 픽셀마다 호출하지 말고 편집 전체를 한 호출로 보낸다.

새 파일을 만드는 도구는 `overwrite`가 `true`가 아니면 기존 파일을 덮어쓰지
않는다. 편집 도구에는 그런 보호 장치가 없다. 파일을 바꾸는 것이 목적이기
때문이다.

색은 `#RRGGBB` 또는 `#RRGGBBAA`. 인덱스 컬러 스프라이트에서는 `5` 같은 맨숫자가
팔레트 인덱스로 쓰인다.

장면은 `stamp_sprites`로 조립한다. 캐릭터와 소품을 각각 자기 파일에 두고,
원하는 위치로 배경에 복사한다. 스탬프는 주어진 순서대로 투명도를 적용해
합성되므로 나중 것이 앞에 놓이고, 원본 파일은 손대지 않는다.

프레임 번호는 1부터다. 레이어 이름은 그룹 안까지 찾으며, 스택 순서상 처음
일치하는 것이 선택된다. 그리기 도구는 기본적으로 첫 이미지 레이어를 쓴다.
그룹은 픽셀을 담지 않으므로 거부된다.

## 빌드

```
go build -o aseprite-mcp.exe .
```

## 설정

MCP 클라이언트에 바이너리를 등록한다. Claude Code라면:

```
claude mcp add aseprite -- C:\path\to\aseprite-mcp.exe
```

또는 `.mcp.json`에 추가한다:

```json
{
  "mcpServers": {
    "aseprite": {
      "command": "C:/path/to/aseprite-mcp.exe"
    }
  }
}
```

### 환경 변수

| 변수 | 의미 |
| --- | --- |
| `ASEPRITE_PATH` | `Aseprite.exe` 경로. 자동 탐지가 실패할 때만 필요하다. |
| `ASEPRITE_MCP_TIMEOUT` | 호출당 타임아웃. Go duration 형식, 예: `30s`. 기본값 `60s`. |

자동 탐지는 흔한 Steam 및 단독 설치 위치를 먼저 살펴보고, 없으면 `PATH`의
`aseprite`로 넘어간다.

## 참고

- `.png` 같은 단일 이미지 포맷은 애니메이션을 담을 수 없다. 여러 프레임짜리
  스프라이트를 그런 포맷으로 내보내면 Aseprite는 요청한 이름 대신 프레임마다
  번호가 붙은 파일(`anim1.png`, `anim2.png`, ...)을 쓴다. 결과의 `files`에 실제
  파일 이름들이 나열되고 `splitIntoSequence`가 설정된다. 하나의 애니메이션
  파일을 원하면 `.gif`를, 한 장의 시트를 원하면 `export_spritesheet`를 쓴다.
- `get_sprite_info`는 팔레트를 최대 32개까지만 나열한다. `paletteSize`는 항상
  실제 개수를 보고하고 `paletteTruncated`가 목록이 잘렸는지 알려준다. 팔레트
  전체는 `get_palette`로 읽는다.

## 테스트

```
go test ./...
```

테스트 스위트는 서버를 빌드해 실제 stdio MCP 세션으로 구동한다. Aseprite가
설치되어 있지 않으면 건너뛴다.
