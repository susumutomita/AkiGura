#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== AkiGura 起動スクリプト ==="

# Check dependencies
command -v go >/dev/null 2>&1 || { echo "エラー: Goがインストールされていません"; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "エラー: Python3がインストールされていません"; exit 1; }

# Check ground-reservation
GROUND_RESERVATION="${GROUND_RESERVATION_PATH:-../ground-reservation}"
if [ ! -d "$GROUND_RESERVATION" ]; then
    echo "エラー: ground-reservationが見つかりません"
    echo "次のコマンドでセットアップしてください:"
    echo "  git clone https://github.com/susumutomita/ground-reservation.git $GROUND_RESERVATION"
    echo "  cd $GROUND_RESERVATION && python3 -m venv venv && source venv/bin/activate && pip install -r requirements.txt"
    exit 1
fi

# Build control-plane
echo "
[ビルド] Control Plane..."
cd control-plane
if [ ! -f akigura-srv ] || [ cmd/srv/main.go -nt akigura-srv ]; then
    go build -o akigura-srv ./cmd/srv
fi
cd ..

# Build worker
echo "[ビルド] Worker..."
cd worker
if [ ! -f akigura-worker ] || [ cmd/worker/main.go -nt akigura-worker ]; then
    go build -o akigura-worker ./cmd/worker
fi
cd ..

# Start control-plane
PORT=${PORT:-8000}
echo "
[起動] Control Plane (port $PORT)..."
cd control-plane
./akigura-srv -listen :$PORT &
SRV_PID=$!
cd ..

# Wait for server to be ready
echo "[待機] サーバー起動待ち..."
for i in {1..30}; do
    if curl -s http://localhost:$PORT/health >/dev/null 2>&1 || curl -s http://localhost:$PORT/ >/dev/null 2>&1; then
        break
    fi
    sleep 0.5
done

echo "
=== AkiGura 起動完了 ==="
echo ""
echo "ダッシュボード: http://localhost:$PORT"
echo ""
echo "Workerを実行するには (別ターミナルで):"
echo "  cd worker && ./akigura-worker -once"
echo ""
echo "Ctrl+C で停止"

# Trap to cleanup
cleanup() {
    echo "
[停止中]..."
    kill $SRV_PID 2>/dev/null || true
    exit 0
}
trap cleanup SIGINT SIGTERM

# Wait for server
wait $SRV_PID
