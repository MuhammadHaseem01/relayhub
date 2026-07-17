package store

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type LedgerQuery struct {
	Page          int
	Limit         int
	SourceType    string
	PaymentStatus string
	ReferenceType string
	DateFrom      string
	DateTo        string
}

type LedgerResponse map[string]any

func (s *Store) ListLedgers(ctx context.Context, query LedgerQuery, currentUserID int) (map[string]any, error) {
	if err := s.SyncMissingLedgers(ctx, currentUserID); err != nil {
		return nil, err
	}
	ledgers, err := s.buildLedgers(ctx, query, currentUserID)
	if err != nil {
		return nil, err
	}
	grouped := groupLedgersByUID(ledgers)
	start := (query.Page - 1) * query.Limit
	if start > len(grouped) {
		start = len(grouped)
	}
	end := start + query.Limit
	if end > len(grouped) {
		end = len(grouped)
	}
	return map[string]any{"success": true, "recordCount": len(grouped), "ledgers": grouped[start:end]}, nil
}

func (s *Store) GetLedgerByID(ctx context.Context, id string, currentUserID int) (map[string]any, error) {
	if err := s.SyncMissingLedgers(ctx, currentUserID); err != nil {
		return nil, err
	}
	ledgers, err := s.buildLedgers(ctx, LedgerQuery{Page: 1, Limit: 100}, currentUserID)
	if err != nil {
		return nil, err
	}
	var uid string
	for _, ledger := range ledgers {
		if fmt.Sprint(ledger["id"]) == id {
			uid, _ = ledger["uid"].(string)
			break
		}
	}
	if uid == "" {
		return nil, ErrNotFound
	}
	records := []map[string]any{}
	total := 0.0
	for _, ledger := range ledgers {
		if ledger["uid"] == uid {
			amount, _ := ledger["amount"].(float64)
			total += amount
			records = append(records, ledger)
		}
	}
	for _, record := range records {
		record["totalAmount"] = total
	}
	return map[string]any{"success": true, "ledger": records}, nil
}

func (s *Store) SyncMissingLedgers(ctx context.Context, currentUserID int) error {
	if err := s.EnsureLedgerTable(ctx); err != nil {
		return err
	}
	tours, err := s.queryMaps(ctx, `SELECT "id", "tourName", "startDate", "fuelDetails", "createdAt", "updatedAt", "createdById" FROM "Tour" WHERE "createdById" = $1`, currentUserID)
	if err != nil {
		return err
	}
	for _, tour := range tours {
		id := intFromAny(tour["id"])
		has, err := s.hasReferenceLedgers(ctx, "tour", id, currentUserID)
		if err != nil {
			return err
		}
		if !has {
			if err := s.ReplaceTourLedgers(ctx, id, currentUserID); err != nil {
				return err
			}
		}
	}
	maintenances, err := s.queryMaps(ctx, `SELECT "id" FROM "VehicleMaintenance" WHERE "createdById" = $1`, currentUserID)
	if err != nil {
		return err
	}
	for _, maintenance := range maintenances {
		id := intFromAny(maintenance["id"])
		has, err := s.hasReferenceLedgers(ctx, "vehicleMaintenance", id, currentUserID)
		if err != nil {
			return err
		}
		if !has {
			if err := s.ReplaceVehicleMaintenanceLedger(ctx, id, currentUserID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) ReplaceTourLedgers(ctx context.Context, tourID int, currentUserID int) error {
	if err := s.EnsureLedgerTable(ctx); err != nil {
		return err
	}
	rows, err := s.queryMaps(ctx, `SELECT "id", "tourName", "startDate", "fuelDetails", "createdAt", "updatedAt", "createdById" FROM "Tour" WHERE "id" = $1 AND "createdById" = $2 LIMIT 1`, tourID, currentUserID)
	if err != nil || len(rows) == 0 {
		return err
	}
	tour := rows[0]
	if err := s.DeleteReferenceLedgers(ctx, "tour", tourID, currentUserID); err != nil {
		return err
	}
	fuelDetails := []map[string]any{}
	if raw, ok := tour["fuelDetails"].(string); ok && raw != "" {
		_ = json.Unmarshal([]byte(raw), &fuelDetails)
	}
	for i, fuel := range fuelDetails {
		sourceName, _ := fuel["petrolPumpName"].(string)
		sourceID := toSourceID(sourceName)
		if sourceName == "" {
			sourceID = toSourceID(fmt.Sprintf("Petrol Pump-%d-%d", tourID, i))
		}
		ownerID := intFromAny(tour["createdById"])
		if ownerID == 0 {
			ownerID = currentUserID
		}
		date, _ := tour["startDate"].(time.Time)
		createdAt, _ := tour["createdAt"].(time.Time)
		updatedAt, _ := tour["updatedAt"].(time.Time)
		if err := s.createLedger(ctx, map[string]any{
			"sourceType": "Petrol Pump", "sourceTypeKey": "petrolPump", "sourceId": sourceID, "sourceName": sourceName,
			"amount": numberFromAny(fuel["amount"]), "paymentStatus": paymentStatus(fuel["paymentStatus"]), "date": date,
			"referenceType": fmt.Sprint(tour["tourName"]), "referenceId": fmt.Sprintf("TOUR-%d-%d", createdAt.UnixMilli(), tourID), "referenceModel": "tour", "referenceRecordId": tourID, "lineIndex": i,
			"createdById": ownerID, "organizationId": fmt.Sprint(ownerID), "createdAt": createdAt, "updatedAt": updatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ReplaceVehicleMaintenanceLedger(ctx context.Context, maintenanceID int, currentUserID int) error {
	if err := s.EnsureLedgerTable(ctx); err != nil {
		return err
	}
	rows, err := s.queryMaps(ctx, `SELECT "id", "vehicleId", "vehicleName", "workService", "date", "amount", "paymentStatus"::TEXT, "vendorWorkshop", "description", "createdAt", "updatedAt", "createdById" FROM "VehicleMaintenance" WHERE "id" = $1 AND "createdById" = $2 LIMIT 1`, maintenanceID, currentUserID)
	if err != nil || len(rows) == 0 {
		return err
	}
	maintenance := rows[0]
	if err := s.DeleteReferenceLedgers(ctx, "vehicleMaintenance", maintenanceID, currentUserID); err != nil {
		return err
	}
	ownerID := intFromAny(maintenance["createdById"])
	if ownerID == 0 {
		ownerID = currentUserID
	}
	date, _ := maintenance["date"].(time.Time)
	createdAt, _ := maintenance["createdAt"].(time.Time)
	updatedAt, _ := maintenance["updatedAt"].(time.Time)
	return s.createLedger(ctx, map[string]any{
		"sourceType": "Workshop", "sourceTypeKey": "workshop", "sourceId": toSourceID(fmt.Sprint(maintenance["vendorWorkshop"])), "sourceName": fmt.Sprint(maintenance["vendorWorkshop"]),
		"amount": numberFromAny(maintenance["amount"]), "paymentStatus": paymentStatus(maintenance["paymentStatus"]), "date": date,
		"referenceType": "vehicleMaintenance", "referenceId": fmt.Sprint(maintenanceID), "referenceModel": "vehicleMaintenance", "referenceRecordId": maintenanceID, "lineIndex": 0,
		"createdById": ownerID, "organizationId": fmt.Sprint(ownerID), "createdAt": createdAt, "updatedAt": updatedAt,
	})
}

func (s *Store) DeleteReferenceLedgers(ctx context.Context, referenceModel string, referenceRecordID int, currentUserID int) error {
	if err := s.EnsureLedgerTable(ctx); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM "Ledger" WHERE "referenceModel" = $1 AND "referenceRecordId" = $2 AND "createdById" = $3`, referenceModel, referenceRecordID, currentUserID)
	return err
}

func (s *Store) EnsureLedgerTable(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS "Ledger" (
		"id" SERIAL PRIMARY KEY,
		"uid" TEXT NOT NULL,
		"sourceType" TEXT NOT NULL,
		"sourceId" TEXT NOT NULL,
		"sourceName" TEXT NOT NULL,
		"amount" DOUBLE PRECISION NOT NULL,
		"paymentStatus" TEXT NOT NULL,
		"date" TIMESTAMP(3) NOT NULL,
		"referenceType" TEXT NOT NULL,
		"referenceId" TEXT NOT NULL,
		"referenceModel" TEXT NOT NULL,
		"referenceRecordId" INTEGER NOT NULL,
		"lineIndex" INTEGER NOT NULL,
		"createdById" INTEGER,
		"organizationId" TEXT,
		"createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
		"updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return err
	}
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS "Ledger_reference_unique" ON "Ledger" ("referenceModel", "referenceRecordId", "lineIndex", "sourceType", "sourceId")`,
		`CREATE INDEX IF NOT EXISTS "Ledger_createdById_idx" ON "Ledger" ("createdById")`,
		`CREATE INDEX IF NOT EXISTS "Ledger_date_idx" ON "Ledger" ("date")`,
		`CREATE INDEX IF NOT EXISTS "Ledger_reference_idx" ON "Ledger" ("referenceModel", "referenceRecordId")`,
	}
	for _, query := range indexes {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) buildLedgers(ctx context.Context, query LedgerQuery, currentUserID int) ([]map[string]any, error) {
	rows, err := s.queryMaps(ctx, `SELECT "id", "uid", "sourceType", "sourceId", "sourceName", "amount", "paymentStatus", "date", "referenceType", "referenceId", "referenceModel", "referenceRecordId", "lineIndex", "createdById", "organizationId", "createdAt", "updatedAt" FROM "Ledger" WHERE "createdById" = $1 ORDER BY "date" DESC, "id" DESC`, currentUserID)
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for _, row := range rows {
		if !matchesLedgerQuery(row, query) {
			continue
		}
		out = append(out, toLedgerResponse(row))
	}
	return out, nil
}

func (s *Store) hasReferenceLedgers(ctx context.Context, referenceModel string, referenceRecordID int, currentUserID int) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM "Ledger" WHERE "referenceModel" = $1 AND "referenceRecordId" = $2 AND "createdById" = $3`, referenceModel, referenceRecordID, currentUserID).Scan(&count)
	return count > 0, err
}

func (s *Store) createLedger(ctx context.Context, input map[string]any) error {
	var id int
	err := s.db.QueryRowContext(ctx, `INSERT INTO "Ledger" ("uid", "sourceType", "sourceId", "sourceName", "amount", "paymentStatus", "date", "referenceType", "referenceId", "referenceModel", "referenceRecordId", "lineIndex", "createdById", "organizationId", "createdAt", "updatedAt") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) ON CONFLICT ("referenceModel", "referenceRecordId", "lineIndex", "sourceType", "sourceId") DO UPDATE SET "amount" = EXCLUDED."amount", "paymentStatus" = EXCLUDED."paymentStatus", "updatedAt" = EXCLUDED."updatedAt" RETURNING "id"`,
		"", input["sourceType"], input["sourceId"], input["sourceName"], input["amount"], input["paymentStatus"], input["date"], input["referenceType"], input["referenceId"], input["referenceModel"], input["referenceRecordId"], input["lineIndex"], input["createdById"], input["organizationId"], input["createdAt"], input["updatedAt"]).Scan(&id)
	if err != nil {
		return err
	}
	uid := toLedgerUID(fmt.Sprint(input["organizationId"]), fmt.Sprint(input["sourceName"]), fmt.Sprint(input["sourceTypeKey"]), intFromAny(input["referenceRecordId"]))
	_, err = s.db.ExecContext(ctx, `UPDATE "Ledger" SET "uid" = $1 WHERE "id" = $2`, uid, id)
	return err
}

func matchesLedgerQuery(row map[string]any, query LedgerQuery) bool {
	if query.SourceType != "" && row["sourceType"] != ledgerSourceDisplay(query.SourceType) {
		return false
	}
	if query.ReferenceType != "" && row["referenceModel"] != query.ReferenceType {
		return false
	}
	if query.PaymentStatus != "" && row["paymentStatus"] != query.PaymentStatus {
		return false
	}
	date, _ := row["date"].(time.Time)
	if query.DateFrom != "" {
		from, err := time.Parse(time.RFC3339, query.DateFrom)
		if err != nil {
			from, _ = time.Parse("2006-01-02", query.DateFrom)
		}
		if !from.IsZero() && date.Before(from) {
			return false
		}
	}
	if query.DateTo != "" {
		to, err := time.Parse(time.RFC3339, query.DateTo)
		if err != nil {
			to, _ = time.Parse("2006-01-02", query.DateTo)
			to = to.Add(24*time.Hour - time.Millisecond)
		}
		if !to.IsZero() && date.After(to) {
			return false
		}
	}
	return true
}

func toLedgerResponse(row map[string]any) map[string]any {
	date, _ := row["date"].(time.Time)
	return map[string]any{
		"id": fmt.Sprint(row["id"]), "uid": row["uid"], "sourceType": row["sourceType"], "sourceId": row["sourceId"], "sourceName": row["sourceName"],
		"amount": numberFromAny(row["amount"]), "paymentStatus": row["paymentStatus"], "date": date.Format("2006-01-02"), "referenceType": row["referenceType"], "referenceId": row["referenceId"],
		"createdBy": fmt.Sprint(row["createdById"]), "organizationId": row["organizationId"], "createdAt": row["createdAt"], "updatedAt": row["updatedAt"], "__v": 0,
	}
}

func groupLedgersByUID(ledgers []map[string]any) []map[string]any {
	order := []string{}
	groups := map[string][]map[string]any{}
	for _, ledger := range ledgers {
		uid, _ := ledger["uid"].(string)
		if _, ok := groups[uid]; !ok {
			order = append(order, uid)
		}
		groups[uid] = append(groups[uid], ledger)
	}
	out := []map[string]any{}
	for _, uid := range order {
		group := groups[uid]
		total := 0.0
		records := []map[string]any{}
		for _, ledger := range group {
			total += numberFromAny(ledger["amount"])
			record := map[string]any{}
			for key, value := range ledger {
				if key != "uid" {
					record[key] = value
				}
			}
			records = append(records, record)
		}
		out = append(out, map[string]any{"uid": uid, "recordCount": len(group), "totalAmount": total, "records": records})
	}
	return out
}

func ledgerSourceDisplay(value string) string {
	if value == "petrolPump" || value == "Petrol Pump" {
		return "Petrol Pump"
	}
	return "workshop"
}

func paymentStatus(value any) string {
	if fmt.Sprint(value) == "Paid" {
		return "Paid"
	}
	return "Pending"
}

var nonAlnumLower = regexp.MustCompile(`[^a-z0-9]+`)
var nonAlnumUpper = regexp.MustCompile(`[^A-Z0-9]+`)

func toSourceID(value string) string {
	out := nonAlnumLower.ReplaceAllString(strings.ToLower(strings.TrimSpace(value)), "-")
	out = strings.Trim(out, "-")
	if out == "" {
		return "unknown"
	}
	return out
}

func toLedgerUID(ownerID string, sourceName string, sourceType string, id int) string {
	return fmt.Sprintf("LEDG-%s-%s-%s-%d", toUIDPart(ownerID), toUIDPart(sourceName), toUIDPart(sourceType), id)
}

func toUIDPart(value string) string {
	out := nonAlnumUpper.ReplaceAllString(strings.ToUpper(strings.TrimSpace(value)), "-")
	out = strings.Trim(out, "-")
	if out == "" {
		return "UNKNOWN"
	}
	return out
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func numberFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		var out float64
		_, _ = fmt.Sscan(v, &out)
		return out
	}
	return 0
}
