#!/usr/bin/env python3
"""Split internal/service/cmdb/servers.go by anchor markers (same package)."""
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1] / "internal" / "service" / "cmdb"
SRC = ROOT / "servers.go"

IMPORTS = {
    "servers_types.go": """package cmdb

import (
	"strings"
	"time"

	"yunshu/internal/model"
)
""",
    "servers_crud.go": """package cmdb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/pkg/sshclient"
	"yunshu/internal/pkg/sshserver"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)
""",
    "servers_groups.go": """package cmdb

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)
""",
    "servers_cloud.go": """package cmdb

import (
	"context"
	"errors"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)
""",
    "servers_connectivity.go": """package cmdb

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	bizerrors "yunshu/internal/pkg/errors"

	"gorm.io/gorm"
)
""",
    "servers_cloud_actions.go": """package cmdb

import (
	"context"
	"errors"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	cryptox "yunshu/internal/pkg/crypto"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/pkg/sshclient"

	"gorm.io/gorm"
)
""",
    "servers_excel.go": """package cmdb

import (
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	bizerrors "yunshu/internal/pkg/errors"
	"yunshu/internal/repository"

	"github.com/xuri/excelize/v2"
)
""",
}

# (filename, start_marker, end_marker) — end_marker is exclusive; empty end = EOF
SPLITS = [
    ("servers_types.go", "type ServerItem struct", "func (s *Service) ListServers"),
    ("servers_crud.go", "func (s *Service) ListServers", "type ServerGroupItem struct"),
    ("servers_groups.go", "type ServerGroupItem struct", "type CloudAccountItem struct"),
    ("servers_cloud.go", "type CloudAccountItem struct", "const ("),
    ("servers_groups.go", "const (", "type ServerTestRequest struct"),  # append
    ("servers_connectivity.go", "type ServerTestRequest struct", "type CloudServerActionRequest struct"),
    ("servers_cloud_actions.go", "type CloudServerActionRequest struct", "func (s *Service) ImportServersFromExcel"),
    ("servers_excel.go", "func (s *Service) ImportServersFromExcel", ""),
]


def find_line(lines: list[str], marker: str, start: int = 0) -> int:
    for i in range(start, len(lines)):
        if lines[i].startswith(marker):
            return i
    raise ValueError(f"marker not found: {marker!r}")


def extract(lines: list[str], start_marker: str, end_marker: str) -> str:
    start = find_line(lines, start_marker)
    if end_marker:
        end = find_line(lines, end_marker, start + 1)
        return "".join(lines[start:end])
    return "".join(lines[start:])


def main() -> None:
    text = SRC.read_text(encoding="utf-8")
    lines = text.splitlines(keepends=True)
    bodies: dict[str, list[str]] = {}
    for name, start, end in SPLITS:
        chunk = extract(lines, start, end)
        bodies.setdefault(name, []).append(chunk)
    for name, chunks in bodies.items():
        body = "".join(chunks)
        (ROOT / name).write_text(IMPORTS[name] + "\n" + body, encoding="utf-8")
        print(f"wrote {name} ({len(body.splitlines())} lines)")
    SRC.unlink()
    print("removed servers.go")


if __name__ == "__main__":
    main()
