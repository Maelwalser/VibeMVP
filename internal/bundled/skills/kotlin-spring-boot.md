# Kotlin + Spring Boot Skill Guide

## Project Layout

```
service-name/
├── build.gradle.kts
├── src/main/kotlin/org/example/
│   ├── controller/   # @RestController handlers
│   ├── service/      # Business logic
│   ├── repository/   # Spring Data repositories
│   ├── model/        # Data classes (DTOs, entities)
│   └── config/       # @ConfigurationProperties, beans
├── src/main/resources/
│   └── application.yml
└── src/test/kotlin/org/example/
```

## build.gradle.kts Boilerplate

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never hardcode version strings.

```kotlin
plugins {
    kotlin("jvm") version "<version>"
    kotlin("plugin.spring") version "<version>"    // all-open for Spring
    kotlin("plugin.jpa") version "<version>"       // no-arg for JPA entities
    id("org.springframework.boot") version "<version>"
    id("io.spring.dependency-management") version "<version>"
}

dependencies {
    implementation("org.springframework.boot:spring-boot-starter-web")
    implementation("org.springframework.boot:spring-boot-starter-data-jpa")
    implementation("org.springframework.boot:spring-boot-starter-validation")
    implementation("com.fasterxml.jackson.module:jackson-module-kotlin")
    implementation("org.jetbrains.kotlin:kotlin-reflect")
    runtimeOnly("org.postgresql:postgresql")
    testImplementation("org.springframework.boot:spring-boot-starter-test")
}
```

## Data Classes as DTOs

```kotlin
// No getters/setters/builders needed — Kotlin data classes do it all
data class CreateUserRequest(
    @field:NotBlank @field:Size(max = 200) val name: String,
    @field:NotNull @field:Email val email: String,
)

data class UserResponse(
    val id: Long,
    val name: String,
    val email: String,
) {
    companion object {
        fun from(entity: UserEntity) = UserResponse(
            id = entity.id!!,
            name = entity.name,
            email = entity.email,
        )
    }
}
```

## Controller Pattern

```kotlin
@RestController
@RequestMapping("/api/users")
@Validated
class UserController(
    private val userService: UserService,  // constructor injection — preferred
) {

    @GetMapping
    fun list(): ResponseEntity<List<UserResponse>> =
        ResponseEntity.ok(userService.listAll())

    @GetMapping("/{id}")
    fun getById(@PathVariable id: Long): ResponseEntity<UserResponse> =
        userService.findById(id)
            ?.let { ResponseEntity.ok(it) }
            ?: ResponseEntity.notFound().build()

    @PostMapping
    fun create(@Valid @RequestBody request: CreateUserRequest): ResponseEntity<UserResponse> {
        val created = userService.create(request)
        return ResponseEntity.status(HttpStatus.CREATED).body(created)
    }

    @DeleteMapping("/{id}")
    fun delete(@PathVariable id: Long): ResponseEntity<Unit> {
        userService.delete(id)
        return ResponseEntity.noContent().build()
    }
}
```

## All-Open Plugin Requirement

Spring requires classes to be open (non-final) for proxying. The `kotlin.plugin.spring` plugin automatically makes these annotations open:
- `@Component`, `@Service`, `@Repository`, `@Controller`, `@RestController`
- `@Configuration`, `@Bean`
- `@Transactional`, `@Cacheable`, `@Async`

**No manual `open` keyword needed** on Spring-annotated classes. JPA entities need `kotlin.plugin.jpa` for the no-arg constructor.

## lateinit vs Nullable Injection

```kotlin
@Service
class UserService(
    private val userRepository: UserRepository,  // preferred: constructor injection
) {
    // Avoid @Autowired field injection
    // If unavoidable (circular deps), use lateinit:
    @Autowired
    private lateinit var otherService: OtherService
    // lateinit = non-null but initialized after construction; throws if accessed before init
    // nullable: private var optionalDep: OptionalDep? = null
}
```

## Coroutines with WebFlux

```kotlin
// build.gradle.kts: add spring-boot-starter-webflux + kotlinx-coroutines-reactor
@RestController
@RequestMapping("/api/users")
class UserController(private val userService: UserService) {

    @GetMapping
    suspend fun list(): List<UserResponse> = userService.listAll()

    @GetMapping("/{id}")
    suspend fun getById(@PathVariable id: Long): ResponseEntity<UserResponse> {
        val user = userService.findById(id) ?: return ResponseEntity.notFound().build()
        return ResponseEntity.ok(user)
    }
}

@Service
class UserService(private val userRepository: UserRepository) {

    suspend fun listAll(): List<UserResponse> = withContext(Dispatchers.IO) {
        userRepository.findAll().map { UserResponse.from(it) }
    }

    // For reactive repositories, use Flow:
    fun streamAll(): Flow<UserResponse> =
        userRepository.findAll().map { UserResponse.from(it) }
}
```

## Extension Functions on Spring APIs

```kotlin
// Useful extension on ResponseEntity
fun <T> T.ok(): ResponseEntity<T> = ResponseEntity.ok(this)
fun <T> T.created(): ResponseEntity<T> = ResponseEntity.status(HttpStatus.CREATED).body(this)

// Extension on ApplicationContext for type-safe bean retrieval
inline fun <reified T : Any> ApplicationContext.getBean(): T = getBean(T::class.java)

// Usage in controller
@GetMapping("/{id}")
fun getById(@PathVariable id: Long): ResponseEntity<UserResponse> =
    userService.findById(id)?.ok() ?: ResponseEntity.notFound().build()
```

## @ConfigurationProperties with Constructor Binding

```kotlin
@ConfigurationProperties(prefix = "app")
data class AppProperties(
    val apiKey: String,
    val database: DatabaseProperties,
)

data class DatabaseProperties(
    val maxPoolSize: Int = 10,
    val connectionTimeout: Duration = Duration.ofSeconds(30),
)

// application.yml
// app:
//   api-key: ${APP_API_KEY}
//   database:
//     max-pool-size: 20
//     connection-timeout: 30s

// Enable in main class or configuration:
@SpringBootApplication
@ConfigurationPropertiesScan
class Application

fun main(args: Array<String>) {
    runApplication<Application>(*args)
}
```

## JPA Entity

```kotlin
@Entity
@Table(name = "users")
class UserEntity(
    @Column(nullable = false, length = 200)
    var name: String,

    @Column(unique = true, nullable = false, length = 320)
    var email: String,

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    val id: Long? = null,  // nullable for pre-persist state
)

// Repository
interface UserRepository : JpaRepository<UserEntity, Long> {
    fun findByEmail(email: String): UserEntity?

    @Query("SELECT u FROM UserEntity u WHERE u.name LIKE %:name%")
    fun searchByName(@Param("name") name: String): List<UserEntity>
}
```

## Global Exception Handling

```kotlin
@RestControllerAdvice
class GlobalExceptionHandler {

    @ExceptionHandler(MethodArgumentNotValidException::class)
    fun handleValidation(ex: MethodArgumentNotValidException): ResponseEntity<Map<String, Any>> {
        val errors = ex.bindingResult.fieldErrors
            .associate { it.field to (it.defaultMessage ?: "invalid") }
        return ResponseEntity.badRequest().body(mapOf("errors" to errors))
    }

    @ExceptionHandler(NoSuchElementException::class)
    fun handleNotFound(ex: NoSuchElementException): ResponseEntity<Map<String, String>> =
        ResponseEntity.status(HttpStatus.NOT_FOUND).body(mapOf("error" to (ex.message ?: "Not found")))

    @ExceptionHandler(Exception::class)
    fun handleGeneric(ex: Exception): ResponseEntity<Map<String, String>> {
        // Log with full stack trace server-side
        return ResponseEntity.internalServerError().body(mapOf("error" to "Internal server error"))
    }
}
```

## Rules

- Use constructor injection in all Spring beans — avoids `lateinit` and makes dependencies explicit.
- Apply `@field:NotBlank` (not `@NotBlank`) on data class properties to target the backing field for Bean Validation.
- Use `kotlin.plugin.spring` to avoid manually marking classes `open` — required for Spring proxying.
- Use `kotlin.plugin.jpa` for JPA entities to generate the required no-arg constructor.
- Prefer `data class` for DTOs and `class` for JPA entities (data classes with mutable `var` and nullable `id` can cause issues with `equals`/`hashCode` in Hibernate).
- Use `@ConfigurationPropertiesScan` at the main class to auto-detect all `@ConfigurationProperties` beans.
- For WebFlux coroutines, wrap blocking calls in `withContext(Dispatchers.IO)`.
- Never use `!!` (non-null assertion) without a clear invariant — prefer `?:` or safe navigation.
