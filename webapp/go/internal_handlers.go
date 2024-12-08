package main

import (
	"database/sql"
	"errors"
	"net/http"
)

var (
	speedByModel map[string]int
)

func init() {
	speedByModel = map[string]int{
		"リラックスシート NEO":      2,
		"エアシェル ライト":         2,
		"チェアエース S":          2,
		"スピンフレーム 01":        2,
		"ベーシックスツール プラス":     2,
		"SitEase":           2,
		"ComfortBasic":      2,
		"EasySit":           2,
		"LiteLine":          2,
		"リラックス座":            2,
		"エルゴクレスト II":        3,
		"フォームライン RX":        3,
		"シェルシート ハイブリッド":     3,
		"リカーブチェア スマート":      3,
		"フレックスコンフォート PRO":   3,
		"ErgoFlex":          3,
		"BalancePro":        3,
		"StyleSit":          3,
		"風雅（ふうが）チェア":        3,
		"AeroSeat":          3,
		"ゲーミングシート NEXUS":    3,
		"プレイスタイル Z":         3,
		"ストリームギア S1":        3,
		"クエストチェア Lite":      3,
		"エアフロー EZ":          3,
		"アルティマシート X":        5,
		"ゼンバランス EX":         5,
		"プレミアムエアチェア ZETA":   5,
		"モーションチェア RISE":     5,
		"インペリアルクラフト LUXE":   5,
		"LuxeThrone":        5,
		"ZenComfort":        5,
		"Infinity Seat":     5,
		"雅楽座":               5,
		"Titanium Line":     5,
		"プロゲーマーエッジ X1":      5,
		"スリムライン GX":         5,
		"フューチャーチェア CORE":    5,
		"シャドウバースト M":        5,
		"ステルスシート ROGUE":     5,
		"ナイトシート ブラックエディション": 7,
		"フューチャーステップ VISION": 7,
		"匠座 PRO LIMITED":    7,
		"ルミナスエアクラウン":        7,
		"エコシート リジェネレイト":     7,
		"ShadowEdition":     7,
		"Phoenix Ultra":     7,
		"匠座（たくみざ）プレミアム":     7,
		"Aurora Glow":       7,
		"Legacy Chair":      7,
		"インフィニティ GEAR V":    7,
		"ゼノバース ALPHA":       7,
		"タイタンフレーム ULTRA":    7,
		"ヴァーチェア SUPREME":    7,
		"オブシディアン PRIME":     7,
	}
}

// このAPIをインスタンス内から一定間隔で叩かせることで、椅子とライドをマッチングさせる
func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// MEMO: 一旦最も待たせているリクエストに適当な空いている椅子マッチさせる実装とする。おそらくもっといい方法があるはず…
	ride := &Ride{}

	tx, err := db.Beginx()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer tx.Rollback()

	if err := tx.GetContext(ctx, ride, `SELECT * FROM rides WHERE chair_id IS NULL ORDER BY created_at LIMIT 1`); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	var chairs [](struct {
		ID        string `db:"id"`
		Model     string `db:"model"`
		Longitude int    `db:"longitude"`
		Latitude  int    `db:"latitude"`
	})
	if err := tx.SelectContext(ctx, &chairs, "SELECT chairs.id as id, chairs.model as model, chair_positions.longitude as longitude, chair_positions.latitude as latitude FROM chairs INNER JOIN chair_positions ON chairs.id = chair_positions.chair_id WHERE is_active = TRUE AND is_available = TRUE"); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(chairs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	chairID := ""
	minTime := float64(1<<31 - 1)
	for _, chair := range chairs {
		t := float64(calculateDistance(ride.PickupLatitude, ride.PickupLongitude, chair.Latitude, chair.Longitude)) / float64(speedByModel[chair.Model])
		if t < minTime {
			chairID = chair.ID
			minTime = t
		}
	}

	if _, err := tx.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", chairID, ride.ID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if _, err := tx.ExecContext(ctx, "UPDATE chairs SET is_available = ? WHERE id = ?", false, chairID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	user := &User{}
	if err := db.GetContext(ctx, user, "SELECT * FROM users WHERE id = ?", ride.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	ride.ChairID = sql.NullString{String: chairID, Valid: true}
	appData, chairData, err := getNotificationInfo(ctx, tx, user, ride)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	err = publishRideUpdateNotification(ctx, user, appData, chairData)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
