# Guía de Integración Backend - Mentor Monitor

## Introducción

Este documento describe cómo integrar el backend Spring Boot con el frontend Vue.js de Mentor Monitor.

## Requisitos Previos

- Spring Boot 2.7+
- Java 11+
- Maven o Gradle
- Conocimientos de REST API

## Configuración Inicial

### 1. CORS Configuration

Configurar CORS en Spring Boot para permitir peticiones desde el frontend:

```java
@Configuration
public class CorsConfig implements WebMvcConfigurer {
    
    @Override
    public void addCorsMappings(CorsRegistry registry) {
        registry.addMapping("/api/**")
                .allowedOrigins("http://localhost:3000", "https://tu-dominio.com")
                .allowedMethods("GET", "POST", "PUT", "DELETE", "OPTIONS")
                .allowedHeaders("*")
                .allowCredentials(true)
                .maxAge(3600);
    }
}
```

### 2. Configuración de Variables de Entorno

El frontend espera que el backend esté en la URL configurada en `.env`:

```env
VITE_API_BASE_URL=http://localhost:8080/api
```

Asegúrese de que su aplicación Spring Boot corra en el puerto 8080.

### 3. Formato de Respuestas

Todas las respuestas deben ser JSON con la siguiente estructura:

#### Respuesta Exitosa (Colección)
```json
{
  "data": [...],
  "total": 100,
  "page": 1,
  "perPage": 10
}
```

#### Respuesta Exitosa (Objeto único)
```json
{
  "id": 1,
  "nombre": "Planta 1",
  "activo": true
}
```

#### Respuesta de Error
```json
{
  "error": true,
  "message": "Mensaje de error descriptivo",
  "code": "ERROR_CODE"
}
```

## Endpoints Requeridos

### Autenticación

#### POST /api/auth/login
Autenticar usuario y obtener token JWT.

**Request:**
```json
{
  "username": "admin",
  "password": "password123"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refreshToken": "refresh_token_here",
  "user": {
    "id": 1,
    "username": "admin",
    "email": "admin@example.com",
    "nombre": "Administrador",
    "rol": "ADMIN"
  }
}
```

#### POST /api/auth/logout
Cerrar sesión (invalidar token).

**Headers:** `Authorization: Bearer {token}`

**Response:**
```json
{
  "success": true
}
```

#### POST /api/auth/refresh
Renovar token expirado.

**Request:**
```json
{
  "refreshToken": "refresh_token_here"
}
```

**Response:**
```json
{
  "token": "new_jwt_token",
  "refreshToken": "new_refresh_token"
}
```

#### GET /api/auth/me
Obtener información del usuario actual.

**Headers:** `Authorization: Bearer {token}`

**Response:**
```json
{
  "id": 1,
  "username": "admin",
  "email": "admin@example.com",
  "nombre": "Administrador",
  "rol": "ADMIN"
}
```

### Usuarios

#### GET /api/usuarios
Listar todos los usuarios.

**Query Params:**
- `page` (int): Número de página (default: 1)
- `perPage` (int): Elementos por página (default: 10)
- `search` (string): Búsqueda por nombre/email

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "nombre": "Administrador",
      "rol": "ADMIN",
      "activo": true
    }
  ],
  "total": 50,
  "page": 1,
  "perPage": 10
}
```

#### GET /api/usuarios/{id}
Obtener un usuario por ID.

**Response:**
```json
{
  "id": 1,
  "username": "admin",
  "email": "admin@example.com",
  "nombre": "Administrador",
  "rol": "ADMIN",
  "activo": true
}
```

#### POST /api/usuarios
Crear nuevo usuario.

**Request:**
```json
{
  "username": "nuevo_usuario",
  "email": "usuario@example.com",
  "nombre": "Nombre Usuario",
  "password": "password123",
  "rol": "OPERATOR"
}
```

**Response:**
```json
{
  "id": 2,
  "username": "nuevo_usuario",
  "email": "usuario@example.com",
  "nombre": "Nombre Usuario",
  "rol": "OPERATOR",
  "activo": true
}
```

#### PUT /api/usuarios/{id}
Actualizar usuario.

**Request:**
```json
{
  "email": "nuevo_email@example.com",
  "nombre": "Nombre Actualizado",
  "rol": "SUPERVISOR"
}
```

**Response:**
```json
{
  "id": 2,
  "username": "nuevo_usuario",
  "email": "nuevo_email@example.com",
  "nombre": "Nombre Actualizado",
  "rol": "SUPERVISOR",
  "activo": true
}
```

#### DELETE /api/usuarios/{id}
Eliminar usuario.

**Response:**
```json
{
  "success": true
}
```

### Plantas

#### GET /api/plantas
Listar todas las plantas.

**Response:**
```json
{
  "data": [
    {
      "id": 1,
      "nombre": "ISM Arequipa",
      "compania": "Industrias San Miguel",
      "lineas": 5,
      "activo": true
    }
  ],
  "total": 12,
  "page": 1,
  "perPage": 10
}
```

#### POST /api/plantas
Crear nueva planta.

**Request:**
```json
{
  "nombre": "Nueva Planta",
  "compania": "Empresa ABC",
  "lineas": 3
}
```

**Response:**
```json
{
  "id": 2,
  "nombre": "Nueva Planta",
  "compania": "Empresa ABC",
  "lineas": 3,
  "activo": true
}
```

#### PUT /api/plantas/{id}
Actualizar planta.

#### DELETE /api/plantas/{id}
Eliminar planta.

### Dashboard

#### GET /api/dashboard/stats
Obtener estadísticas del dashboard.

**Response:**
```json
{
  "totalPlantas": 12,
  "totalLineas": 48,
  "totalDispositivos": 156,
  "alarmasActivas": 3,
  "eficienciaPromedio": 87.5,
  "produccionHoy": 12450
}
```

#### GET /api/dashboard/reportes
Obtener datos para gráficos de reportes.

**Query Params:**
- `startDate` (string): Fecha inicio (ISO 8601)
- `endDate` (string): Fecha fin (ISO 8601)

**Response:**
```json
{
  "data": [
    {
      "fecha": "2025-11-18",
      "valor": 8500,
      "tipo": "Programada"
    }
  ]
}
```

### Análisis

#### GET /api/analisis/general
Obtener datos de análisis general.

**Query Params:**
- `startDate` (string): Fecha inicio
- `endDate` (string): Fecha fin
- `plantId` (int): ID de planta
- `lineaId` (int): ID de línea
- `dispositivoId` (int): ID de dispositivo
- `variableId` (int): ID de variable

**Response:**
```json
{
  "data": [
    {
      "timestamp": "2025-11-18T10:30:00Z",
      "value": 4250.5,
      "status": "normal"
    }
  ]
}
```

#### GET /api/analisis/energia
Datos de análisis de energía.

#### GET /api/analisis/produccion
Datos de análisis de producción.

#### GET /api/analisis/pareto
Datos para gráfico de Pareto.

**Response:**
```json
{
  "data": [
    {
      "categoria": "Falla Mecánica",
      "frecuencia": 45,
      "porcentaje": 35
    }
  ]
}
```

## Seguridad

### JWT Token

El frontend envía el token JWT en cada petición autenticada:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Configuración de Seguridad en Spring

```java
@Configuration
@EnableWebSecurity
public class SecurityConfig extends WebSecurityConfigurerAdapter {
    
    @Override
    protected void configure(HttpSecurity http) throws Exception {
        http
            .cors()
            .and()
            .csrf().disable()
            .authorizeRequests()
            .antMatchers("/api/auth/**").permitAll()
            .antMatchers("/api/**").authenticated()
            .and()
            .sessionManagement()
            .sessionCreationPolicy(SessionCreationPolicy.STATELESS);
        
        http.addFilterBefore(jwtAuthenticationFilter(), 
                             UsernamePasswordAuthenticationFilter.class);
    }
}
```

### Interceptor JWT

El frontend intercepta respuestas 401 y automáticamente intenta renovar el token.

## Manejo de Fechas

Todas las fechas deben estar en formato ISO 8601:

```
2025-11-18T14:30:00Z
```

### Configuración en Spring Boot

```java
@Configuration
public class JacksonConfig {
    
    @Bean
    public ObjectMapper objectMapper() {
        ObjectMapper mapper = new ObjectMapper();
        mapper.registerModule(new JavaTimeModule());
        mapper.disable(SerializationFeature.WRITE_DATES_AS_TIMESTAMPS);
        return mapper;
    }
}
```

## Paginación

El frontend espera paginación en este formato:

**Request:**
```
GET /api/usuarios?page=2&perPage=20
```

**Response:**
```json
{
  "data": [...],
  "total": 150,
  "page": 2,
  "perPage": 20,
  "totalPages": 8
}
```

## Códigos de Estado HTTP

Utilice los códigos de estado HTTP apropiados:

- `200 OK`: Operación exitosa
- `201 Created`: Recurso creado exitosamente
- `400 Bad Request`: Datos inválidos
- `401 Unauthorized`: No autenticado
- `403 Forbidden`: Sin permisos
- `404 Not Found`: Recurso no encontrado
- `500 Internal Server Error`: Error del servidor

## Ejemplo de Controller Spring Boot

```java
@RestController
@RequestMapping("/api/plantas")
@CrossOrigin(origins = "*")
public class PlantaController {
    
    @Autowired
    private PlantaService plantaService;
    
    @GetMapping
    public ResponseEntity<?> getAll(
            @RequestParam(defaultValue = "1") int page,
            @RequestParam(defaultValue = "10") int perPage) {
        
        Page<Planta> plantas = plantaService.findAll(
            PageRequest.of(page - 1, perPage));
        
        Map<String, Object> response = new HashMap<>();
        response.put("data", plantas.getContent());
        response.put("total", plantas.getTotalElements());
        response.put("page", page);
        response.put("perPage", perPage);
        
        return ResponseEntity.ok(response);
    }
    
    @GetMapping("/{id}")
    public ResponseEntity<?> getById(@PathVariable Long id) {
        Planta planta = plantaService.findById(id);
        if (planta == null) {
            return ResponseEntity.notFound().build();
        }
        return ResponseEntity.ok(planta);
    }
    
    @PostMapping
    public ResponseEntity<?> create(@RequestBody PlantaDTO dto) {
        Planta planta = plantaService.create(dto);
        return ResponseEntity.status(HttpStatus.CREATED).body(planta);
    }
    
    @PutMapping("/{id}")
    public ResponseEntity<?> update(
            @PathVariable Long id, 
            @RequestBody PlantaDTO dto) {
        Planta planta = plantaService.update(id, dto);
        return ResponseEntity.ok(planta);
    }
    
    @DeleteMapping("/{id}")
    public ResponseEntity<?> delete(@PathVariable Long id) {
        plantaService.delete(id);
        Map<String, Boolean> response = new HashMap<>();
        response.put("success", true);
        return ResponseEntity.ok(response);
    }
}
```

## Testing

Use Postman o cURL para probar los endpoints antes de conectar con el frontend:

```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'
```

## Modo Mock

El frontend incluye un modo mock que no requiere backend:

```env
VITE_USE_MOCKS=true
```

Active este modo para desarrollar el frontend sin backend disponible.

## Solución de Problemas Comunes

### Error CORS
- Verificar configuración de CORS en Spring
- Permitir headers `Authorization` y `Content-Type`

### Token Expirado
- El frontend renueva automáticamente tokens expirados
- Implementar endpoint `/api/auth/refresh`

### Formato de Fecha Incorrecto
- Usar ISO 8601: `2025-11-18T14:30:00Z`
- Configurar Jackson correctamente

### Error 401 en todas las peticiones
- Verificar que el token se envíe en headers
- Revisar configuración de Spring Security

## Contacto y Soporte

Para dudas sobre la integración, revise:
- Documentación de API en `/docs/api-contracts/`
- Colección Postman en `/docs/postman/`
- Código fuente de servicios en `/src/api/services/`
