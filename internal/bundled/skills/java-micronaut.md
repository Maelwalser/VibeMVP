# Java + Micronaut Skill Guide

## Project Layout

```
service-name/
├── build.gradle (or pom.xml)
├── src/main/java/org/example/
│   ├── controller/   # @Controller HTTP handlers
│   ├── service/      # @Singleton business logic
│   ├── repository/   # Data access
│   ├── model/        # DTOs and domain types
│   └── config/       # @ConfigurationProperties
├── src/main/resources/
│   └── application.yml
└── src/test/java/org/example/
```

## build.gradle Boilerplate

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never hardcode version strings.

```groovy
plugins {
    id("io.micronaut.application") version "<version>"
}

micronaut {
    runtime("netty")
    testRuntime("junit5")
}

dependencies {
    annotationProcessor("io.micronaut:micronaut-http-validation")
    annotationProcessor("io.micronaut.validation:micronaut-validation-processor")
    implementation("io.micronaut:micronaut-http-server-netty")
    implementation("io.micronaut.data:micronaut-data-jdbc")
    implementation("io.micronaut.validation:micronaut-validation")
    runtimeOnly("ch.qos.logback:logback-classic")
    testImplementation("io.micronaut.test:micronaut-test-junit5")
}
```

## Controller Pattern

```java
@Controller("/users")
public class UserController {

    private final UserService userService;

    // Constructor injection — Micronaut resolves at compile time
    public UserController(UserService userService) {
        this.userService = userService;
    }

    @Get
    public List<UserDTO> list() {
        return userService.listAll();
    }

    @Get("/{id}")
    public HttpResponse<UserDTO> getById(@PathVariable Long id) {
        return userService.findById(id)
            .map(HttpResponse::ok)
            .orElse(HttpResponse.notFound());
    }

    @Post
    @Status(HttpStatus.CREATED)
    public UserDTO create(@Body @Valid CreateUserRequest request) {
        return userService.create(request);
    }

    @Delete("/{id}")
    @Status(HttpStatus.NO_CONTENT)
    public void delete(@PathVariable Long id) {
        userService.delete(id);
    }
}
```

## Compile-Time DI

```java
@Singleton
public class UserService {

    private final UserRepository userRepository;

    public UserService(UserRepository userRepository) {
        this.userRepository = userRepository;
    }

    public List<UserDTO> listAll() {
        return userRepository.findAll().stream()
            .map(UserDTO::from)
            .collect(Collectors.toList());
    }

    @Transactional
    public UserDTO create(CreateUserRequest request) {
        User user = new User(request.name(), request.email());
        userRepository.save(user);
        return UserDTO.from(user);
    }
}
```

## Configuration Properties

```java
@ConfigurationProperties("app")
public class AppConfiguration {
    private String apiKey;
    private int maxConnections = 10;

    // getters and setters required
    public String getApiKey() { return apiKey; }
    public void setApiKey(String apiKey) { this.apiKey = apiKey; }
    public int getMaxConnections() { return maxConnections; }
    public void setMaxConnections(int maxConnections) { this.maxConnections = maxConnections; }
}
```

```yaml
# application.yml
app:
  api-key: ${APP_API_KEY}
  max-connections: 20

datasources:
  default:
    url: ${DATABASE_URL}
    driver-class-name: org.postgresql.Driver
    dialect: POSTGRES
```

## AOP with @Around Interceptors

```java
// 1. Define the binding annotation
@Retention(RetentionPolicy.RUNTIME)
@Target({ElementType.METHOD})
@Around
public @interface Timed {}

// 2. Implement the interceptor
@Singleton
@InterceptorBean(Timed.class)
public class TimedInterceptor implements MethodInterceptor<Object, Object> {
    private static final Logger log = LoggerFactory.getLogger(TimedInterceptor.class);

    @Override
    public Object intercept(MethodInvocationContext<Object, Object> context) {
        long start = System.currentTimeMillis();
        try {
            return context.proceed();
        } finally {
            long elapsed = System.currentTimeMillis() - start;
            log.info("method={} durationMs={}", context.getMethodName(), elapsed);
        }
    }
}

// 3. Apply to service methods
@Singleton
public class UserService {
    @Timed
    public List<UserDTO> listAll() { ... }
}
```

## Error Handling

```java
@Produces
@Singleton
@Requires(classes = {ConstraintViolationException.class, ExceptionHandler.class})
public class ValidationExceptionHandler implements ExceptionHandler<ConstraintViolationException, HttpResponse<Map<String, Object>>> {

    @Override
    public HttpResponse<Map<String, Object>> handle(HttpRequest request, ConstraintViolationException ex) {
        Map<String, Object> errors = new LinkedHashMap<>();
        ex.getConstraintViolations().forEach(v ->
            errors.put(v.getPropertyPath().toString(), v.getMessage())
        );
        return HttpResponse.badRequest(Map.of("errors", errors));
    }
}
```

## GraalVM Native Image

Micronaut generates GraalVM metadata at compile time — no manual reflection config needed for:
- `@Controller`, `@Singleton`, `@Inject`, `@ConfigurationProperties`
- Micronaut Data repositories

For third-party classes that require reflection:
```java
@ReflectiveAccess  // Micronaut-specific, marks for native image
public class ThirdPartyDTO { ... }
```

Build native:
```bash
./gradlew nativeCompile
./gradlew nativeRun
```

## Testing with @MicronautTest

```java
@MicronautTest
class UserControllerTest {

    @Inject
    @Client("/")
    HttpClient client;

    @Inject
    UserRepository userRepository;

    @Test
    void testListUsers() {
        HttpResponse<List<UserDTO>> response = client.toBlocking()
            .exchange(HttpRequest.GET("/users"), Argument.listOf(UserDTO.class));

        assertEquals(HttpStatus.OK, response.status());
        assertNotNull(response.body());
    }

    @Test
    void testCreateUser() {
        CreateUserRequest req = new CreateUserRequest("Alice", "alice@example.com");
        HttpResponse<UserDTO> response = client.toBlocking()
            .exchange(HttpRequest.POST("/users", req), UserDTO.class);

        assertEquals(HttpStatus.CREATED, response.status());
        assertEquals("Alice", response.body().name());
    }
}
```

## Health Check

```yaml
# application.yml
endpoints:
  health:
    enabled: true
    sensitive: false
```

```java
@Singleton
public class DatabaseHealthIndicator implements HealthIndicator {
    @Override
    public Publisher<HealthResult> getResult() {
        return Mono.just(HealthResult.builder("database", HealthStatus.UP).build());
    }
}
```

## Rules

- Use constructor injection exclusively — Micronaut resolves it at compile time, avoiding runtime reflection.
- Annotate all injection points with `@Inject` only when field/setter injection is unavoidable (prefer constructors).
- Use `@Body` for request bodies, `@PathVariable` for path params, `@QueryValue` for query params.
- Validate inputs with `@Valid` on controller parameters; Micronaut runs Bean Validation automatically.
- All configuration must go through `@ConfigurationProperties` or `@Value` — never hardcode values.
- Use `@Transactional` on service methods, not controllers.
- For native image, prefer Micronaut Data over raw JDBC to minimize reflection surface.
