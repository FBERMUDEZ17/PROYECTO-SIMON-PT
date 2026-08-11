package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// TAREA: servicio de autenticación (register/login) — email normalizado a
// minúsculas, password hasheado con bcrypt, JWT de 24h vía auth.GenerateToken.
var (
	ErrEmailAlreadyExists = errors.New("el email ya está registrado")
	ErrInvalidCredentials = errors.New("email o password incorrectos")
)

// RoleUser y RoleAdmin son los roles válidos de un usuario.
// TAREA (sensores/IoT): base del enmascarado de device ids por rol.
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

// User es la representación pública de un usuario (sin el hash del password).
type User struct {
	ID        int64
	Email     string
	Name      string
	Role      string
	CreatedAt string
}

// Service implementa el registro y login de usuarios, emitiendo JWTs manuales.
type Service struct {
	db        *sql.DB
	jwtSecret []byte
	now       func() time.Time // inyectable para tests
}

// NewService crea un Service. jwtSecret no debe estar vacío.
func NewService(database *sql.DB, jwtSecret []byte) *Service {
	return &Service{
		db:        database,
		jwtSecret: jwtSecret,
		now:       time.Now,
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ParseToken verifica un JWT emitido por este Service (mismo secreto) y
// devuelve sus claims. Es la puerta de entrada que usa el middleware HTTP.
func (s *Service) ParseToken(token string) (*Claims, error) {
	return VerifyToken(s.jwtSecret, token, s.now())
}

// Register hashea el password, crea el usuario y devuelve un JWT.
// TAREA: "registro exitoso" / "registro con email duplicado" (ver tests).
func (s *Service) Register(email, password, name string) (User, string, error) {
	email = normalizeEmail(email) // TAREA: evita duplicados por diferencia de mayúsculas.

	var exists int
	err := s.db.QueryRow(`SELECT 1 FROM users WHERE email = ?`, email).Scan(&exists)
	if err == nil {
		return User{}, "", ErrEmailAlreadyExists
	}
	if err != sql.ErrNoRows {
		return User{}, "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, "", err
	}

	now := s.now()
	res, err := s.db.Exec(
		`INSERT INTO users (email, password_hash, name, role, created_at) VALUES (?, ?, ?, ?, ?)`,
		email, string(hash), name, RoleUser, now.UTC().Format(time.RFC3339),
	)
	if err != nil {
		// Salvaguarda ante condición de carrera: la constraint UNIQUE de la
		// tabla es la que garantiza no-duplicados en concurrencia real.
		if strings.Contains(err.Error(), "UNIQUE") {
			return User{}, "", ErrEmailAlreadyExists
		}
		return User{}, "", err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return User{}, "", err
	}

	user := User{ID: id, Email: email, Name: name, Role: RoleUser, CreatedAt: now.UTC().Format(time.RFC3339)}

	token, err := GenerateToken(s.jwtSecret, strconv.FormatInt(id, 10), email, name, RoleUser, now)
	if err != nil {
		return User{}, "", err
	}

	return user, token, nil
}

// SetRole cambia el rol de un usuario existente (p.ej. para promover un
// admin desde una tarea de seed/migración). No hay endpoint HTTP público
// para esto: se gestiona fuera de banda, deliberadamente.
// TAREA (sensores/IoT): habilita probar el enmascarado admin vs no-admin.
func (s *Service) SetRole(email, role string) error {
	if role != RoleUser && role != RoleAdmin {
		return fmt.Errorf("rol inválido: %q", role)
	}
	email = normalizeEmail(email)
	res, err := s.db.Exec(`UPDATE users SET role = ? WHERE email = ?`, role, email)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ListUsers devuelve todos los usuarios registrados (sin el hash del
// password), ordenados por email.
//
// TAREA: alimenta el selector de "propietario" al crear un vehículo nuevo
// (solo admin) — reutiliza la tabla users existente, no agrega columnas.
func (s *Service) ListUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT id, email, name, role, created_at FROM users ORDER BY email`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// GetUserByID devuelve el usuario con ese id, o sql.ErrNoRows si no existe.
// Usado para validar el propietario elegido al crear un vehículo.
func (s *Service) GetUserByID(id int64) (User, error) {
	var u User
	err := s.db.QueryRow(
		`SELECT id, email, name, role, created_at FROM users WHERE id = ?`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt)
	if err != nil {
		return User{}, err
	}
	return u, nil
}

// Login verifica las credenciales y devuelve un JWT si son válidas.
// TAREA: "login exitoso" / "password incorrecto" / "email inexistente" /
// "email en mayúsculas" (ver tests) — todos cubiertos abajo.
func (s *Service) Login(email, password string) (User, string, error) {
	email = normalizeEmail(email) // TAREA: permite loguear con "EMAIL@X.COM" igual que "email@x.com".

	var (
		id           int64
		passwordHash string
		name         string
		role         string
		createdAt    string
	)
	err := s.db.QueryRow(
		`SELECT id, password_hash, name, role, created_at FROM users WHERE email = ?`, email,
	).Scan(&id, &passwordHash, &name, &role, &createdAt)

	if errors.Is(err, sql.ErrNoRows) {
		// Mensaje genérico: no revelamos si el email existe o no.
		return User{}, "", ErrInvalidCredentials
	}
	if err != nil {
		return User{}, "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return User{}, "", ErrInvalidCredentials
	}

	now := s.now()
	user := User{ID: id, Email: email, Name: name, Role: role, CreatedAt: createdAt}

	token, err := GenerateToken(s.jwtSecret, strconv.FormatInt(id, 10), email, name, role, now)
	if err != nil {
		return User{}, "", err
	}

	return user, token, nil
}
