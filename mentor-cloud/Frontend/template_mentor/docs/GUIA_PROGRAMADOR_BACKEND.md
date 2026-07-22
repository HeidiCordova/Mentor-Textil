# 🚀 Guía Completa para el Programador Backend

## 📋 Índice

1. [Visión General](#visión-general)
2. [Estructura del Frontend](#estructura-del-frontend)
3. [Contratos de API](#contratos-de-api)
4. [Implementación por Módulos](#implementación-por-módulos)
5. [Seguridad y Autenticación](#seguridad-y-autenticación)
6. [Base de Datos Sugerida](#base-de-datos-sugerida)
7. [Endpoints Prioritarios](#endpoints-prioritarios)
8. [Formatos de Respuesta](#formatos-de-respuesta)
9. [Manejo de Errores](#manejo-de-errores)
10. [Testing y Validación](#testing-y-validación)
11. [Checklist de Implementación](#checklist-de-implementación)

---

## 🎯 Visión General

### Objetivo
Desarrollar un backend RESTful en **Spring Boot** que se integre con el frontend Vue.js de Mentor Monitor, un sistema de monitoreo industrial para gestión de plantas, líneas de producción, análisis OEE y reportes.

### Tecnologías Recomendadas
- **Framework:** Spring Boot 3.x
- **Base de Datos:** PostgreSQL o MySQL
- **Seguridad:** Spring Security + JWT
- **Documentación:** Swagger/OpenAPI
- **ORM:** Spring Data JPA
- **Validación:** Bean Validation
- **Logs:** SLF4J + Logback

### URL Base del API
```
http://localhost:8080/api
```

---

## 📂 Estructura del Frontend

El frontend está organizado en módulos funcionales:

```
src/
├── api/                    # Cliente HTTP y servicios
│   ├── client.js          # Axios con interceptors JWT
│   ├── endpoints.js       # URLs centralizadas
│   └── services/          # 6 servicios principales
│       ├── auth.service.js
│       ├── dashboard.service.js
│       ├── plant.service.js
│       ├── analysis.service.js
│       ├── company.service.js
│       └── user.service.js
│
├── modules/               # 12+ módulos funcionales
│   ├── auth/             # Autenticación
│   ├── dashboard/        # Dashboard principal
│   ├── administracion/   # Turnos, productos, calendarización
│   ├── configuracion/    # Empresas, plantas, usuarios, árbol paradas
│   ├── analisis/         # Análisis general
│   ├── analisis-produccion/  # OEE, Pareto, tiempo real
│   ├── analisis-energia/     # Consumo eléctrico
│   ├── analisis-avanzado/    # Consultas personalizadas
│   ├── alarmas/          # Gestión de alarmas
│   ├── compromisos/      # Seguimiento de compromisos
│   ├── datos-recibidos/  # Datos de dispositivos
│   └── reportes/         # Generación de reportes
│
└── stores/               # Estado global (Pinia)
    ├── auth.js          # Usuario y tokens
    ├── filters.js       # Filtros globales
    └── ui.js            # Estado de UI
```

**📖 Documentos Relacionados:**
- `docs/ARQUITECTURA_FRONTEND.md` - Arquitectura completa
- `docs/GUIA_SERVICIOS_API.md` - Servicios y uso detallado

---

## 📜 Contratos de API

Los contratos JSON Schema están en `docs/api-contracts/`:

| Archivo | Endpoints | Descripción |
|---------|-----------|-------------|
| `auth.json` | `/api/auth/*` | Login, logout, refresh token |
| `dashboard.json` | `/api/dashboard/*` | Estadísticas, KPIs, gráficos |
| `plantas.json` | `/api/plantas/*` | CRUD de plantas y líneas |
| `analisis.json` | `/api/analisis/*` | Análisis de producción, energía, OEE |
| `empresas.json` | `/api/empresas/*` | CRUD de empresas |
| `usuarios.json` | `/api/usuarios/*` | CRUD de usuarios y roles |

**Total de Endpoints:** 40+

---

## 🏗️ Implementación por Módulos

### 1. Módulo de Autenticación (PRIORIDAD ALTA)

#### Endpoints
```
POST   /api/auth/login      - Autenticar usuario
POST   /api/auth/logout     - Cerrar sesión
POST   /api/auth/refresh    - Renovar JWT token
GET    /api/auth/me         - Usuario actual
```

#### Entidades
```java
@Entity
@Table(name = "usuarios")
public class Usuario {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;
    
    @Column(unique = true, nullable = false)
    private String username;
    
    @Column(nullable = false)
    private String password; // Hasheado con BCrypt
    
    @Column(unique = true, nullable = false)
    private String email;
    
    private String nombre;
    private String apellido;
    
    @Enumerated(EnumType.STRING)
    private Rol rol; // ADMIN, SUPERVISOR, OPERATOR, VIEWER
    
    @ManyToOne
    @JoinColumn(name = "empresa_id")
    private Empresa empresa;
    
    private Boolean activo = true;
    
    @Column(name = "ultimo_acceso")
    private LocalDateTime ultimoAcceso;
    
    @Column(name = "fecha_creacion")
    private LocalDateTime fechaCreacion;
}

public enum Rol {
    ADMIN,      // Acceso total
    SUPERVISOR, // Gestión y reportes
    OPERATOR,   // Operación básica
    VIEWER      // Solo lectura
}
```

#### Flujo JWT
1. Usuario envía `{ username, password }`
2. Backend valida credenciales
3. Backend genera:
   - **Access Token** (JWT, 1 hora de vida)
   - **Refresh Token** (JWT, 7 días de vida)
4. Frontend guarda tokens en `localStorage`
5. Cada request incluye: `Authorization: Bearer {accessToken}`
6. Si token expira (401), frontend llama `/auth/refresh`

#### Configuración Spring Security
```java
@Configuration
@EnableWebSecurity
public class SecurityConfig {
    
    @Bean
    public SecurityFilterChain filterChain(HttpSecurity http) throws Exception {
        http
            .csrf().disable()
            .cors().and()
            .authorizeHttpRequests()
                .requestMatchers("/api/auth/login", "/api/auth/refresh").permitAll()
                .requestMatchers("/api/**").authenticated()
            .and()
            .sessionManagement()
                .sessionCreationPolicy(SessionCreationPolicy.STATELESS)
            .and()
            .addFilterBefore(jwtAuthenticationFilter(), 
                UsernamePasswordAuthenticationFilter.class);
        return http.build();
    }
    
    @Bean
    public PasswordEncoder passwordEncoder() {
        return new BCryptPasswordEncoder();
    }
}
```

---

### 2. Módulo de Dashboard (PRIORIDAD ALTA)

#### Endpoints
```
GET /api/dashboard/stats           - Estadísticas generales
GET /api/dashboard/production      - Datos de producción por hora
GET /api/dashboard/efficiency      - Métricas OEE
```

#### Response Example - Stats
```json
{
  "totalPlantas": 12,
  "totalLineas": 48,
  "totalDispositivos": 156,
  "alarmasActivas": 3,
  "eficienciaPromedio": 87.5,
  "produccionHoy": 15240
}
```

#### Response Example - Production
```json
{
  "data": [
    { "hora": "00:00", "unidades": 450, "meta": 500 },
    { "hora": "01:00", "unidades": 420, "meta": 500 },
    ...
  ],
  "totalDia": 15240,
  "promedioPorHora": 635
}
```

#### Response Example - Efficiency
```json
{
  "oee": {
    "disponibilidad": 92,
    "rendimiento": 85,
    "calidad": 95,
    "oeeTotal": 74.3
  },
  "desglose": {
    "tiempoTotal": 1440,
    "tiempoOperativo": 1325,
    "tiempoParadas": 115,
    "unidadesProducidas": 15240,
    "unidadesDefectuosas": 762
  }
}
```

#### Query Parameters
- `startDate`: Fecha inicio (ISO 8601)
- `endDate`: Fecha fin (ISO 8601)
- `plantId`: ID de planta (opcional)
- `lineId`: ID de línea (opcional)

---

### 3. Módulo de Plantas (PRIORIDAD ALTA)

#### Endpoints
```
GET    /api/plantas                 - Listar plantas (paginado)
GET    /api/plantas/{id}            - Detalle de planta
POST   /api/plantas                 - Crear planta
PUT    /api/plantas/{id}            - Actualizar planta
DELETE /api/plantas/{id}            - Eliminar planta
GET    /api/plantas/{id}/lineas     - Líneas de una planta
GET    /api/plantas/{id}/dispositivos - Dispositivos de una planta
```

#### Entidades
```java
@Entity
@Table(name = "plantas")
public class Planta {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;
    
    @Column(nullable = false)
    private String nombre;
    
    private String codigo;
    private String ubicacion;
    private String descripcion;
    
    @ManyToOne
    @JoinColumn(name = "empresa_id")
    private Empresa empresa;
    
    @OneToMany(mappedBy = "planta", cascade = CascadeType.ALL)
    private List<Linea> lineas;
    
    private Boolean activo = true;
    private LocalDateTime fechaCreacion;
}

@Entity
@Table(name = "lineas")
public class Linea {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;
    
    @Column(nullable = false)
    private String nombre;
    
    private String codigo;
    
    @ManyToOne
    @JoinColumn(name = "planta_id")
    private Planta planta;
    
    @OneToMany(mappedBy = "linea")
    private List<Dispositivo> dispositivos;
    
    private Integer capacidadHora; // Unidades por hora
    private Boolean activo = true;
}
```

#### Response con Paginación
```json
{
  "data": [
    {
      "id": 1,
      "nombre": "Planta Norte",
      "codigo": "PN-001",
      "ubicacion": "Lima, Perú",
      "empresa": {
        "id": 1,
        "nombre": "Empresa XYZ"
      },
      "totalLineas": 12,
      "activo": true
    }
  ],
  "total": 12,
  "page": 1,
  "limit": 20,
  "totalPages": 1
}
```

---

### 4. Módulo de Análisis (PRIORIDAD MEDIA)

#### Endpoints
```
GET /api/analisis/produccion  - Análisis de producción
GET /api/analisis/energia     - Análisis energético
GET /api/analisis/oee         - Análisis OEE detallado
GET /api/analisis/pareto      - Gráfico de Pareto (paradas)
```

#### Response Example - Pareto
```json
{
  "data": [
    {
      "categoria": "Falta de Material",
      "frecuencia": 45,
      "porcentaje": 35.2,
      "acumulado": 35.2
    },
    {
      "categoria": "Avería Mecánica",
      "frecuencia": 32,
      "porcentaje": 25.0,
      "acumulado": 60.2
    }
  ],
  "totalParadas": 128,
  "tiempoTotalParadas": 450
}
```

---

### 5. Módulo de Empresas (PRIORIDAD MEDIA)

#### Endpoints
```
GET    /api/empresas         - Listar empresas
GET    /api/empresas/{id}    - Detalle de empresa
POST   /api/empresas         - Crear empresa
PUT    /api/empresas/{id}    - Actualizar empresa
DELETE /api/empresas/{id}    - Eliminar empresa
```

#### Entidad
```java
@Entity
@Table(name = "empresas")
public class Empresa {
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;
    
    @Column(nullable = false)
    private String nombre;
    
    @Column(unique = true, length = 11)
    private String ruc;
    
    private String direccion;
    private String telefono;
    private String email;
    
    @OneToMany(mappedBy = "empresa")
    private List<Planta> plantas;
    
    @OneToMany(mappedBy = "empresa")
    private List<Usuario> usuarios;
    
    private Boolean activo = true;
    private LocalDateTime fechaCreacion;
}
```

---

### 6. Módulo de Usuarios (PRIORIDAD MEDIA)

#### Endpoints
```
GET    /api/usuarios              - Listar usuarios (paginado)
GET    /api/usuarios/{id}         - Detalle de usuario
POST   /api/usuarios              - Crear usuario
PUT    /api/usuarios/{id}         - Actualizar usuario
DELETE /api/usuarios/{id}         - Eliminar usuario
GET    /api/usuarios/roles        - Listar roles disponibles
```

#### Validaciones
- Username: único, 3-50 caracteres, alfanumérico
- Email: único, formato válido
- Password: mínimo 8 caracteres
- Rol: debe ser válido (ADMIN, SUPERVISOR, OPERATOR, VIEWER)

---

## 🔐 Seguridad y Autenticación

### CORS Configuration
```java
@Configuration
public class CorsConfig {
    
    @Bean
    public CorsFilter corsFilter() {
        UrlBasedCorsConfigurationSource source = 
            new UrlBasedCorsConfigurationSource();
        
        CorsConfiguration config = new CorsConfiguration();
        config.setAllowCredentials(true);
        config.addAllowedOrigin("http://localhost:3000");
        config.addAllowedOrigin("http://localhost:3001");
        config.addAllowedHeader("*");
        config.addAllowedMethod("*");
        
        source.registerCorsConfiguration("/api/**", config);
        return new CorsFilter(source);
    }
}
```

### JWT Utility Class
```java
@Component
public class JwtUtil {
    
    @Value("${jwt.secret}")
    private String secret;
    
    @Value("${jwt.expiration}")
    private Long expiration; // 1 hora
    
    public String generateToken(UserDetails userDetails) {
        Map<String, Object> claims = new HashMap<>();
        return Jwts.builder()
            .setClaims(claims)
            .setSubject(userDetails.getUsername())
            .setIssuedAt(new Date())
            .setExpiration(new Date(System.currentTimeMillis() + expiration * 1000))
            .signWith(SignatureAlgorithm.HS512, secret)
            .compact();
    }
    
    public Boolean validateToken(String token, UserDetails userDetails) {
        final String username = extractUsername(token);
        return (username.equals(userDetails.getUsername()) && !isTokenExpired(token));
    }
}
```

---

## 🗄️ Base de Datos Sugerida

### Esquema Principal

```sql
-- Empresas
CREATE TABLE empresas (
    id BIGSERIAL PRIMARY KEY,
    nombre VARCHAR(100) NOT NULL,
    ruc VARCHAR(11) UNIQUE,
    direccion VARCHAR(200),
    telefono VARCHAR(20),
    email VARCHAR(100),
    activo BOOLEAN DEFAULT TRUE,
    fecha_creacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Usuarios
CREATE TABLE usuarios (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    nombre VARCHAR(100),
    apellido VARCHAR(100),
    rol VARCHAR(20) NOT NULL,
    empresa_id BIGINT REFERENCES empresas(id),
    activo BOOLEAN DEFAULT TRUE,
    ultimo_acceso TIMESTAMP,
    fecha_creacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Plantas
CREATE TABLE plantas (
    id BIGSERIAL PRIMARY KEY,
    nombre VARCHAR(100) NOT NULL,
    codigo VARCHAR(50),
    ubicacion VARCHAR(200),
    descripcion TEXT,
    empresa_id BIGINT REFERENCES empresas(id),
    activo BOOLEAN DEFAULT TRUE,
    fecha_creacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Líneas de Producción
CREATE TABLE lineas (
    id BIGSERIAL PRIMARY KEY,
    nombre VARCHAR(100) NOT NULL,
    codigo VARCHAR(50),
    planta_id BIGINT REFERENCES plantas(id),
    capacidad_hora INTEGER,
    activo BOOLEAN DEFAULT TRUE,
    fecha_creacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Dispositivos
CREATE TABLE dispositivos (
    id BIGSERIAL PRIMARY KEY,
    nombre VARCHAR(100) NOT NULL,
    codigo VARCHAR(50),
    tipo VARCHAR(50),
    linea_id BIGINT REFERENCES lineas(id),
    activo BOOLEAN DEFAULT TRUE,
    fecha_creacion TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Producción (Datos históricos)
CREATE TABLE produccion (
    id BIGSERIAL PRIMARY KEY,
    linea_id BIGINT REFERENCES lineas(id),
    fecha_hora TIMESTAMP NOT NULL,
    unidades_producidas INTEGER NOT NULL,
    unidades_defectuosas INTEGER DEFAULT 0,
    tiempo_operativo INTEGER, -- minutos
    tiempo_paradas INTEGER,   -- minutos
    fecha_registro TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Paradas
CREATE TABLE paradas (
    id BIGSERIAL PRIMARY KEY,
    linea_id BIGINT REFERENCES lineas(id),
    categoria VARCHAR(100),
    subcategoria VARCHAR(100),
    descripcion TEXT,
    fecha_inicio TIMESTAMP NOT NULL,
    fecha_fin TIMESTAMP,
    duracion_minutos INTEGER,
    usuario_id BIGINT REFERENCES usuarios(id)
);

-- Índices para mejorar performance
CREATE INDEX idx_produccion_linea_fecha ON produccion(linea_id, fecha_hora);
CREATE INDEX idx_paradas_linea_fecha ON paradas(linea_id, fecha_inicio);
CREATE INDEX idx_usuarios_empresa ON usuarios(empresa_id);
CREATE INDEX idx_plantas_empresa ON plantas(empresa_id);
```

---

## ⚡ Endpoints Prioritarios

### Fase 1 - Críticos (Semana 1-2)
- [ ] POST `/api/auth/login`
- [ ] POST `/api/auth/refresh`
- [ ] GET `/api/auth/me`
- [ ] GET `/api/dashboard/stats`
- [ ] GET `/api/plantas` (lista)
- [ ] GET `/api/plantas/{id}`

### Fase 2 - Importantes (Semana 3-4)
- [ ] GET `/api/dashboard/production`
- [ ] GET `/api/dashboard/efficiency`
- [ ] POST `/api/plantas`
- [ ] PUT `/api/plantas/{id}`
- [ ] GET `/api/empresas`
- [ ] GET `/api/usuarios`

### Fase 3 - Complementarios (Semana 5-6)
- [ ] GET `/api/analisis/produccion`
- [ ] GET `/api/analisis/oee`
- [ ] GET `/api/analisis/pareto`
- [ ] CRUD completo de usuarios
- [ ] CRUD completo de empresas

---

## 📤 Formatos de Respuesta

### Respuesta Exitosa
```json
{
  "id": 1,
  "nombre": "Planta Norte",
  "activo": true
}
```

### Respuesta con Paginación
```json
{
  "data": [...],
  "total": 100,
  "page": 1,
  "limit": 20,
  "totalPages": 5
}
```

### Respuesta de Error
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Los datos proporcionados no son válidos",
    "details": [
      {
        "field": "email",
        "message": "El email es requerido"
      },
      {
        "field": "password",
        "message": "La contraseña debe tener al menos 8 caracteres"
      }
    ]
  },
  "timestamp": "2025-11-19T10:30:00Z",
  "path": "/api/usuarios"
}
```

---

## ⚠️ Manejo de Errores

### Códigos HTTP Estándar

| Código | Significado | Uso |
|--------|-------------|-----|
| 200 | OK | Operación exitosa (GET, PUT) |
| 201 | Created | Recurso creado (POST) |
| 204 | No Content | Eliminación exitosa (DELETE) |
| 400 | Bad Request | Validación fallida |
| 401 | Unauthorized | No autenticado o token inválido |
| 403 | Forbidden | No tiene permisos |
| 404 | Not Found | Recurso no existe |
| 409 | Conflict | Conflicto (ej: username duplicado) |
| 500 | Internal Server Error | Error del servidor |

### Exception Handler Global
```java
@RestControllerAdvice
public class GlobalExceptionHandler {
    
    @ExceptionHandler(MethodArgumentNotValidException.class)
    public ResponseEntity<ErrorResponse> handleValidationExceptions(
            MethodArgumentNotValidException ex) {
        
        List<FieldError> errors = ex.getBindingResult()
            .getFieldErrors()
            .stream()
            .map(error -> new FieldError(
                error.getField(), 
                error.getDefaultMessage()
            ))
            .collect(Collectors.toList());
        
        ErrorResponse response = new ErrorResponse(
            "VALIDATION_ERROR",
            "Los datos proporcionados no son válidos",
            errors
        );
        
        return ResponseEntity.badRequest().body(response);
    }
    
    @ExceptionHandler(ResourceNotFoundException.class)
    public ResponseEntity<ErrorResponse> handleResourceNotFound(
            ResourceNotFoundException ex) {
        
        ErrorResponse response = new ErrorResponse(
            "NOT_FOUND",
            ex.getMessage(),
            null
        );
        
        return ResponseEntity.status(HttpStatus.NOT_FOUND).body(response);
    }
}
```

---

## 🧪 Testing y Validación

### Postman Collection
Crear colección con:
- Variables de entorno (`{{baseUrl}}`, `{{token}}`)
- Pre-request script para token refresh
- Tests automáticos de status code

### Unit Tests
```java
@SpringBootTest
@AutoConfigureMockMvc
class PlantaControllerTest {
    
    @Autowired
    private MockMvc mockMvc;
    
    @Test
    void deberiaListarPlantas() throws Exception {
        mockMvc.perform(get("/api/plantas")
                .header("Authorization", "Bearer " + token))
            .andExpect(status().isOk())
            .andExpect(jsonPath("$.data").isArray());
    }
}
```

---

## ✅ Checklist de Implementación

### Configuración Inicial
- [ ] Proyecto Spring Boot 3.x creado
- [ ] Base de datos PostgreSQL/MySQL configurada
- [ ] application.properties con variables correctas
- [ ] Dependencias agregadas (Spring Security, JWT, JPA, etc.)
- [ ] CORS configurado para localhost:3000 y 3001

### Seguridad
- [ ] JWT token generation implementado
- [ ] Refresh token implementado
- [ ] Password hashing con BCrypt
- [ ] Filtro de autenticación JWT
- [ ] Manejo de expiración de tokens

### Endpoints Críticos
- [ ] POST /api/auth/login funcional
- [ ] GET /api/dashboard/stats funcional
- [ ] GET /api/plantas funcional
- [ ] GET /api/plantas/{id} funcional

### Validaciones
- [ ] Bean Validation en DTOs
- [ ] Validación de unicidad (username, email)
- [ ] Validación de relaciones (FK)
- [ ] Manejo global de excepciones

### Documentación
- [ ] Swagger/OpenAPI configurado
- [ ] README con instrucciones
- [ ] Variables de entorno documentadas
- [ ] Postman collection creada

### Testing
- [ ] Unit tests para servicios
- [ ] Integration tests para controllers
- [ ] Tests de seguridad (401, 403)
- [ ] Tests de validación (400)

---

## 📞 Soporte

Para dudas sobre la integración:
1. Revisar `docs/BACKEND_INTEGRATION.md`
2. Consultar contratos en `docs/api-contracts/`
3. Verificar ejemplos en mock data `src/api/mocks/`

---

**Última actualización:** Noviembre 2025  
**Versión del Template Frontend:** 2.0  
**Autor:** Equipo Mentor Monitor
