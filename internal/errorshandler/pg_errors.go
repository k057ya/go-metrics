package errorshandler

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresErrorClassifier struct {
	BaseClassifier
}

func NewPostgresErrorClassifier() *PostgresErrorClassifier {
	return &PostgresErrorClassifier{
		BaseClassifier: NewBaseClassifier(ClassifyPgError),
	}
}

func ClassifyPgError(err error) Classification {
	if err == nil {
		return NotApplicable
	}
	if errors.Is(err, sql.ErrNoRows) {
		return NonRetriable
	}

	// Ошибки коммита транзакции обрабатываются отдельно
	// тк на этой стадии обычные сетевые ошибки не должны
	// быть `Retriable` (например увеличение счетчика counter
	//  может произойти дважды)
	var commitError *CommitError
	if errors.As(err, &commitError) {
		return classifyCommitError(commitError.Err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr != nil {
		// Коды ошибок PostgreSQL: https://www.postgresql.org/docs/current/errcodes-appendix.html
		switch pgErr.Code {

		// Класс 08 - Ошибки соединения
		case pgerrcode.ConnectionException,
			pgerrcode.ConnectionDoesNotExist,
			pgerrcode.ConnectionFailure,
			pgerrcode.SQLClientUnableToEstablishSQLConnection,
			pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection:
			return Retriable
		case pgerrcode.TransactionResolutionUnknown: // 08007
			return NonRetriable

		// Класс 40 - Откат транзакции
		case pgerrcode.TransactionRollback, // 40000
			pgerrcode.SerializationFailure, // 40001
			pgerrcode.DeadlockDetected:     // 40P01
			return Retriable

		case pgerrcode.StatementCompletionUnknown: // 40003
			return NonRetriable

		// Класс 57 - Ошибка оператора
		case pgerrcode.CannotConnectNow: // 57P03
			return Retriable
		}
		return NonRetriable
	}

	if errors.Is(err, driver.ErrBadConn) {
		return Retriable
	}
	return NotApplicable
}

type CommitError struct {
	Err error
}

func (e *CommitError) Error() string { return fmt.Sprintf("commit: %v", e.Err) }
func (e *CommitError) Unwrap() error { return e.Err }

func classifyCommitError(err error) Classification {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr != nil {
		switch pgErr.Code {
		case pgerrcode.TransactionRollback, pgerrcode.SerializationFailure, pgerrcode.DeadlockDetected:
			return Retriable
		}
	}
	return NonRetriable
}
