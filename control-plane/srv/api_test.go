package srv

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"srv.exe.dev/db/dbgen"
)

func TestFacilityHandlers(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test_api.sqlite3")
	t.Cleanup(func() { os.Remove(tempDB) })

	server, err := New(tempDB, "test-hostname")
	if err != nil {
		t.Fatalf("サーバー初期化に失敗すべきではない: %v", err)
	}

	ctx := context.Background()

	t.Run("施設更新で URL とステータスが更新されるべき", func(t *testing.T) {
		facility, err := server.Queries.CreateFacility(ctx, dbgen.CreateFacilityParams{
			ID:           "facility-update-1",
			Name:         "旧施設",
			Municipality: "旧自治体",
			ScraperType:  "old-scraper",
			Url:          "https://old.example.com",
		})
		if err != nil {
			t.Fatalf("施設作成に失敗すべきではない: %v", err)
		}

		payload := map[string]any{
			"name":         "新施設",
			"municipality": "新自治体",
			"scraper_type": "new-scraper",
			"url":          "https://new.example.com",
			"enabled":      false,
		}
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("更新リクエストの JSON 生成に失敗すべきではない: %v", err)
		}

		req := httptest.NewRequest(http.MethodPut, "/admin/api/facilities/"+facility.ID, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("id", facility.ID)
		w := httptest.NewRecorder()

		server.HandleUpdateFacility(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("更新レスポンスは 200 を返すべき: %d", w.Code)
		}

		updated, err := server.Queries.GetFacility(ctx, facility.ID)
		if err != nil {
			t.Fatalf("更新後の施設取得に失敗すべきではない: %v", err)
		}
		if updated.Name != "新施設" {
			t.Fatalf("施設名は更新されるべき: %s", updated.Name)
		}
		if updated.Municipality != "新自治体" {
			t.Fatalf("自治体名は更新されるべき: %s", updated.Municipality)
		}
		if updated.ScraperType != "new-scraper" {
			t.Fatalf("スクレイパータイプは更新されるべき: %s", updated.ScraperType)
		}
		if updated.Url != "https://new.example.com" {
			t.Fatalf("URL は更新されるべき: %s", updated.Url)
		}
		if updated.Enabled != 0 {
			t.Fatalf("ステータスは無効になるべき: %d", updated.Enabled)
		}
	})

	t.Run("施設削除で対象レコードが削除されるべき", func(t *testing.T) {
		facility, err := server.Queries.CreateFacility(ctx, dbgen.CreateFacilityParams{
			ID:           "facility-delete-1",
			Name:         "削除対象施設",
			Municipality: "削除自治体",
			ScraperType:  "delete-scraper",
			Url:          "https://delete.example.com",
		})
		if err != nil {
			t.Fatalf("施設作成に失敗すべきではない: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, "/admin/api/facilities/"+facility.ID, nil)
		req.SetPathValue("id", facility.ID)
		w := httptest.NewRecorder()

		server.HandleDeleteFacility(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("削除レスポンスは 204 を返すべき: %d", w.Code)
		}

		_, err = server.Queries.GetFacility(ctx, facility.ID)
		if err == nil || !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("削除後は対象レコードが取得できないべき: %v", err)
		}
	})
}
