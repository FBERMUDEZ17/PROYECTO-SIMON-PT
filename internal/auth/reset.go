package auth

// TAREA: "crear un restaurador de contraseña" — flujo de dos pasos:
// RequestPasswordReset genera un token de un solo uso (expira en 30 min);
// ResetPassword lo canjea por un password nuevo. No hay servicio de email
// configurado en el proyecto: el token se loguea en la consola del
// servidor (ver httpapi.ForgotPassword) en vez de mandarse por email —
// documentado explícitamente como decisión de desarrollo, no de producción.

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ResetTokenTTL es la vigencia del token de restablecimiento.
const ResetTokenTTL = 30 * time.Minute

var (
	ErrInvalidResetToken = errors.New("token de restablecimiento inválido o ya usado")
	ErrResetTokenExpired = errors.New("token de restablecimiento expirado")
)

// RequestPasswordReset genera un token de reset si existe un usuario con
// ese email. found es false si el email no está registrado — el caller
// HTTP debe responder igual en ambos casos (no revelar qué emails
// existen), pero usa found para decidir si loguear el token o no.
func (s *Service) RequestPasswordReset(email string) (token string, found bool, err error) {
	email = normalizeEmail(email)

	var userID int64
	err = s.db.QueryRow(`SELECT id FROM users WHERE email = ?`, email).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	token, err = generateResetToken()
	if err != nil {
		return "", false, err
	}

	expiresAt := s.now().Add(ResetTokenTTL).UTC().Format(time.RFC3339)
	_, err = s.db.Exec(
		`INSERT INTO password_resets (user_id, token, expires_at) VALUES (?, ?, ?)`,
		userID, token, expiresAt,
	)
	if err != nil {
		return "", false, err
	}

	return token, true, nil
}

func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ResetPassword valida el token (existe, no usado, no expirado) y
// actualiza el password del usuario dueño, marcando el token como usado
// (de un solo uso).
func (s *Service) ResetPassword(token, newPassword string) error {
	var (
		id        int64
		userID    int64
		expiresAt string
		usedAt    sql.NullString
	)
	err := s.db.QueryRow(
		`SELECT id, user_id, expires_at, used_at FROM password_resets WHERE token = ?`, token,
	).Scan(&id, &userID, &expiresAt, &usedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidResetToken
	}
	if err != nil {
		return err
	}
	if usedAt.Valid {
		return ErrInvalidResetToken
	}

	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return err
	}
	if s.now().After(expiry) {
		return ErrResetTokenExpired
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op si ya se hizo Commit

	if _, err := tx.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, string(hash), userID); err != nil {
		return err
	}
	usedNow := s.now().UTC().Format(time.RFC3339)
	if _, err := tx.Exec(`UPDATE password_resets SET used_at = ? WHERE id = ?`, usedNow, id); err != nil {
		return err
	}

	return tx.Commit()
}
