package domain

import "time"

type Rol struct {
	ID          int       `json:"id"`
	Nombre      string    `json:"nombre"`
	Descripcion string    `json:"descripcion,omitempty"`
	Permisos    []string  `json:"permisos"`
	CreadoEn    time.Time `json:"creado_en"`
}

type Empresa struct {
	ID            int       `json:"id"`
	Nombre        string    `json:"nombre"`
	RUC           string    `json:"ruc"`
	Direccion     string    `json:"direccion,omitempty"`
	Telefono      string    `json:"telefono,omitempty"`
	Email         string    `json:"email,omitempty"`
	Responsable   string    `json:"responsable,omitempty"`
	Estado        bool      `json:"estado"`
	CreadoEn      time.Time `json:"creado_en"`
	ActualizadoEn time.Time `json:"-"`
}

type Usuario struct {
	ID            int       `json:"id"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	Nombre        string    `json:"nombre"`
	Apellido      string    `json:"apellido"`
	PasswordHash  string    `json:"-"`
	RolID         *int      `json:"rol_id,omitempty"`
	Rol           string    `json:"rol,omitempty"`
	EmpresaID     *int      `json:"empresa_id,omitempty"`
	PlantaID      *int      `json:"planta_id,omitempty"`
	Activo        bool      `json:"activo"`
	CreadoEn      time.Time `json:"-"`
	ActualizadoEn time.Time `json:"-"`
}

type RefreshToken struct {
	ID        string    `json:"id"`
	UsuarioID int       `json:"usuario_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Revocado  bool      `json:"revocado"`
	CreadoEn  time.Time `json:"creado_en"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refreshToken"`
	User         UserProfile `json:"user"`
}

type UserProfile struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Plant     string `json:"plant"`
	Company   string `json:"company"`
	EmpresaID *int   `json:"empresa_id,omitempty"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}
