package postgre

import (
	"context"
	"fmt"
	"time"

	"analytics/internal/domain/models"
	"analytics/internal/ports"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var _ ports.AnalyticsStorage = (*PostgreDatabase)(nil)

// GetResults берёт из БД данные по результатам задач
func (db *PostgreDatabase) GetResults(ctx context.Context) (models.ResultsReport, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "GetReport")
	defer span.End()

	query := `SELECT result, count(*) FROM taskresult GROUP BY result;`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return models.ResultsReport{}, fmt.Errorf("get result report was failed: %w", err)
	}

	var res models.ResultsReport
	for rows.Next() {
		var result bool
		var count int
		err = rows.Scan(
			&result,
			&count,
		)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return models.ResultsReport{}, fmt.Errorf("get result report was failed: %w", err)
		}

		if result {
			res.ApprovedTasks = count
		} else {
			res.DeclinedTasks = count
		}
	}
	if rows.Err() != nil {
		span.SetStatus(codes.Error, rows.Err().Error())
		return models.ResultsReport{}, fmt.Errorf("get result report was failed: %w", rows.Err())
	}

	return res, nil
}

// SetResult записывает в БД данные о результатах задач
func (db *PostgreDatabase) SetResult(ctx context.Context, data models.ResultData) error {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "SetResult")
	defer span.End()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("set result was failed: %w", err)
	}
	// nolint
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO taskresult (taskid, result) VALUES($1,$2);`, data.TaskID, data.Result)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("set result was failed: %w", err)
	}

	return tx.Commit(ctx)
}

// GetDuration берёт из БД данные о времени согласования задачи
func (db *PostgreDatabase) GetDuration(ctx context.Context, eventType string) (map[string]time.Duration, error) {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "GetDuration")
	defer span.End()

	// Запрос данных о времени, потраченном на согласование
	query := `SELECT a.taskid, SUM(endtime-starttime) AS time
		FROM approvetime AS a
		INNER JOIN taskresult AS t
		ON a.taskid = t.taskid
		WHERE eventtype = $1
		GROUP BY a.taskid;`
	rows, err := db.Pool.Query(ctx, query, eventType)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("get duration data was failed: %w", err)
	}

	res := make(map[string]time.Duration)
	for rows.Next() {
		var id string
		var duration time.Duration
		err = rows.Scan(
			&id,
			&duration,
		)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return nil, fmt.Errorf("get duration data was failed: %w", err)
		}
		res[id] = duration
	}
	if rows.Err() != nil {
		span.SetStatus(codes.Error, rows.Err().Error())
		return nil, fmt.Errorf("get duration data was failed: %w", rows.Err())
	}

	return res, nil
}

// SetTimestamp записывает в БД данные о времени события
func (db *PostgreDatabase) SetTimestamp(ctx context.Context, data models.TimestampData) error {
	ctx, span := otel.Tracer(models.TracerName).Start(ctx, "SetTimestamp")
	defer span.End()

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("set timestamp was failed: %w", err)
	}
	// nolint
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO approvetime
	(taskid, approver, eventtype, starttime, endtime) VALUES($1,$2,$3,$4,$5);`,
		data.TaskID, data.Approver, data.EventType, data.Start, data.End)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("set timestamp was failed: %w", err)
	}

	return tx.Commit(ctx)
}
