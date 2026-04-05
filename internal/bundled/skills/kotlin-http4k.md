# Kotlin + http4k Skill Guide

## Project Layout

```
service-name/
├── build.gradle.kts
├── src/main/kotlin/org/example/
│   ├── App.kt          # Server entry point
│   ├── routes/         # Route definitions (HttpHandler functions)
│   ├── filters/        # Filter (middleware) definitions
│   ├── lenses/         # Lens definitions for request/response
│   ├── service/        # Business logic
│   └── model/          # Domain types and DTOs
└── src/test/kotlin/org/example/
```

## build.gradle.kts Boilerplate

> ⚠ **Use versions from the "Dependency & API Reference" section** of your task prompt — never hardcode version strings.

```kotlin
plugins {
    kotlin("jvm") version "<version>"
}

val http4kVersion = "<version>"  // from Dependency & API Reference

dependencies {
    implementation(platform("org.http4k:http4k-bom:$http4kVersion"))
    implementation("org.http4k:http4k-core")
    implementation("org.http4k:http4k-server-jetty")          // or undertow/netty
    implementation("org.http4k:http4k-format-jackson")        // JSON serialization
    implementation("org.http4k:http4k-contract")              // optional: OpenAPI
    testImplementation(kotlin("test"))
    testImplementation("org.http4k:http4k-testing-kotest")
}
```

## Core Concept: HttpHandler

```kotlin
// HttpHandler is just a type alias: (Request) -> Response
// Everything is a function — no annotations, no magic

val helloHandler: HttpHandler = { request: Request ->
    Response(Status.OK).body("Hello, ${request.uri.path}!")
}
```

## Server Setup

```kotlin
// App.kt
fun main() {
    val port = System.getenv("PORT")?.toInt() ?: 8080
    val app = buildApp()
    val server = app.asServer(Jetty(port))
    server.start()
    println("Server started on port $port")
}

fun buildApp(): HttpHandler {
    val userService = UserService()
    val userRoutes = userRoutes(userService)

    return routes(
        "/api/users" bind userRoutes,
        "/health" bind GET to { Response(Status.OK).body("ok") },
    ).withFilter(requestLoggingFilter())
     .withFilter(errorHandlingFilter())
}
```

## Router with routes()

```kotlin
// routes/UserRoutes.kt
fun userRoutes(userService: UserService): HttpHandler = routes(
    "/" bind GET to listUsers(userService),
    "/" bind POST to createUser(userService),
    "/{id}" bind GET to getUser(userService),
    "/{id}" bind DELETE to deleteUser(userService),
)

val idLens = Path.long().of("id")
val userBodyLens = Jackson.autoBody<CreateUserRequest>().toLens()
val userResponseLens = Jackson.autoBody<UserResponse>().toLens()

fun listUsers(service: UserService): HttpHandler = {
    val users = service.listAll()
    userResponseListLens(users, Response(Status.OK))
}

fun getUser(service: UserService): HttpHandler = { request ->
    val id = idLens(request)
    val user = service.findById(id)
        ?: return@HttpHandler Response(Status.NOT_FOUND)
    userResponseLens(user, Response(Status.OK))
}

fun createUser(service: UserService): HttpHandler = { request ->
    val body = userBodyLens(request)
    val created = service.create(body)
    userResponseLens(created, Response(Status.CREATED))
}

fun deleteUser(service: UserService): HttpHandler = { request ->
    val id = idLens(request)
    service.delete(id)
    Response(Status.NO_CONTENT)
}
```

## Lenses for Type-Safe Parameter Extraction

```kotlin
// Lenses extract typed values from requests/responses — throw LensFailure if missing/invalid
val idPath = Path.long().of("id")
val pageQuery = Query.int().defaulting("page", 0)
val limitQuery = Query.int().defaulting("limit", 20)
val nameQuery = Query.string().optional("name")

// Required query param — throws 400 if missing
val requiredSearch = Query.string().required("q")

// Header lens
val authHeader = Header.required("Authorization")

// Body lens using Jackson auto-marshalling
val createRequestLens = Jackson.autoBody<CreateUserRequest>().toLens()
val responseBodyLens = Jackson.autoBody<UserResponse>().toLens()

// Usage in handler:
fun searchUsers(service: UserService): HttpHandler = { request ->
    val page = pageQuery(request)
    val limit = limitQuery(request)
    val name = nameQuery(request)  // null if absent
    val results = service.search(name, page, limit)
    userListResponseLens(results, Response(Status.OK))
}
```

## Filter for Middleware

```kotlin
// Filter wraps a handler: Filter = (HttpHandler) -> HttpHandler

fun requestLoggingFilter(): Filter = Filter { next ->
    { request ->
        val start = System.currentTimeMillis()
        val response = next(request)
        val elapsed = System.currentTimeMillis() - start
        println("${request.method} ${request.uri} -> ${response.status.code} (${elapsed}ms)")
        response
    }
}

fun errorHandlingFilter(): Filter = Filter { next ->
    { request ->
        try {
            next(request)
        } catch (e: LensFailure) {
            Response(Status.BAD_REQUEST).body("""{"error": "${e.message}"}""")
        } catch (e: Exception) {
            Response(Status.INTERNAL_SERVER_ERROR).body("""{"error": "Internal server error"}""")
        }
    }
}

fun authFilter(tokenValidator: (String) -> Boolean): Filter = Filter { next ->
    { request ->
        val token = Header.optional("Authorization")(request)
        if (token != null && tokenValidator(token.removePrefix("Bearer "))) {
            next(request)
        } else {
            Response(Status.UNAUTHORIZED)
        }
    }
}

// Apply filter to specific routes:
val protectedRoutes = routes(
    "/admin" bind adminRoutes(),
).withFilter(authFilter { token -> validateJwt(token) })
```

## JSON Serialization (Jackson)

```kotlin
// Use Jackson.autoBody<T>() for data class marshalling
data class CreateUserRequest(val name: String, val email: String)
data class UserResponse(val id: Long, val name: String, val email: String)

val requestLens = Jackson.autoBody<CreateUserRequest>().toLens()
val responseLens = Jackson.autoBody<UserResponse>().toLens()

fun createUser(service: UserService): HttpHandler = { request ->
    val body = requestLens(request)           // deserialize
    val result = service.create(body)
    responseLens(result, Response(Status.CREATED))  // serialize
}
```

## Server Backends

```kotlin
// Jetty (default, production-ready)
app.asServer(Jetty(8080)).start()

// Undertow (high-throughput, non-blocking)
app.asServer(Undertow(8080)).start()

// Netty (async, reactive)
app.asServer(Netty(8080)).start()

// For testing (no network, in-process)
val testClient = app.asClient()  // MockHttpHandler — see Testing section
```

## Testing with MockHttpHandler

```kotlin
class UserRoutesTest {

    private val mockService = MockUserService()
    private val app = userRoutes(mockService)

    @Test
    fun `GET slash returns list of users`() {
        val response = app(Request(Method.GET, "/"))

        assertEquals(Status.OK, response.status)
        val users = Jackson.autoBody<List<UserResponse>>().toLens()(response)
        assertTrue(users.isNotEmpty())
    }

    @Test
    fun `POST slash creates user`() {
        val request = Request(Method.POST, "/")
            .with(Jackson.autoBody<CreateUserRequest>().toLens() of CreateUserRequest("Alice", "alice@example.com"))

        val response = app(request)

        assertEquals(Status.CREATED, response.status)
    }

    @Test
    fun `GET slash id returns 404 for missing user`() {
        val response = app(Request(Method.GET, "/99999"))
        assertEquals(Status.NOT_FOUND, response.status)
    }
}
```

## Rules

- `HttpHandler` is `(Request) -> Response` — embrace it. No annotations, no reflection, no magic.
- Use `routes()` to compose handlers; use `bind` with HTTP methods to attach handlers to paths.
- Use lenses for all parameter extraction — they provide type safety and automatic 400 responses on failure.
- Use `Filter` for cross-cutting concerns (logging, auth, error handling) and compose with `.withFilter()`.
- Test handlers directly by calling them as functions — no HTTP client or server needed for unit tests.
- Use `Jackson.autoBody<T>().toLens()` for request/response body marshalling with data classes.
- Choose server backend in `main()` only — keep handler code server-agnostic.
- Never throw exceptions in handlers for expected error cases — return `Response(Status.NOT_FOUND)` etc.
