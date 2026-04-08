package deps

// LibraryAPIDocs holds the exported API surface of commonly-misused libraries.
// Injected into agent prompts to prevent hallucinated types/functions.
//
// Each entry is keyed by a lowercase technology name that matches against
// the task's technology stack. Add new entries here to extend prompt guidance
// for additional libraries without touching the prompt builder logic.
var LibraryAPIDocs = map[string]string{
	"pgxmock": `## pgxmock/v4 API Reference (github.com/pashagolub/pgxmock/v4)

Creating a mock pool:
  mock, err := pgxmock.NewPool()
  // Returns an interface-satisfying mock. The concrete type is UNEXPORTED.

Exported mock interface type:
  pgxmock.PgxPoolIface   ← the CORRECT type for mock pool in function signatures and struct fields
  pgxmock.PgxConnIface   ← the CORRECT type for mock connection

DO NOT reference any of these — they do not exist:
  pgxmock.PgxPoolMock    ← WRONG — use pgxmock.PgxPoolIface
  pgxmock.PgxPool        ← WRONG — use pgxmock.PgxPoolIface
  pgxmock.PgxMock        ← WRONG — use pgxmock.PgxPoolIface
  pgxmock.MockPool       ← WRONG — use pgxmock.PgxPoolIface
  pgxmock.Pool           ← WRONG — use pgxmock.PgxPoolIface
  mock.ExpectQueryRow()  ← WRONG, does not exist — use mock.ExpectQuery() instead
                            (ExpectQuery handles BOTH Query() and QueryRow() calls)
  pgx.PgError            ← WRONG, does not exist in pgx v5 — use pgconn.PgError instead
                            (PgError moved to the pgconn sub-package in v5)

CRITICAL — mock type in function signatures:
  When you need to use the mock type explicitly (e.g. in test helper function parameters,
  struct fields, or table-driven test setup functions), use pgxmock.PgxPoolIface:
    CORRECT:  func setupMock(mock pgxmock.PgxPoolIface)  // PgxPoolIface is the exported interface
    CORRECT:  mock, err := pgxmock.NewPool()              // inferred type also works
    WRONG:    func setupMock(mock pgxmock.PgxPool)        // PgxPool does NOT exist

CRITICAL — pgconn import path:
  pgxmock/v4 uses types from "github.com/jackc/pgx/v5/pgconn" (the v5 sub-package).
  NEVER import standalone "github.com/jackc/pgconn" — that is the v4-era package with
  DIFFERENT Go types that have the SAME names. Mixing them causes "wrong type for method" errors.

Correct usage pattern:
  import (
      "github.com/jackc/pgx/v5"         // pgx.Rows, pgx.Row
      "github.com/jackc/pgx/v5/pgconn"  // pgconn.CommandTag — MUST be v5 sub-package
  )

  // Define your own interface for the pool:
  type DBTX interface {
      Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
      Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
      QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
  }

  // In tests:
  mock, err := pgxmock.NewPool()
  repo := NewRepository(mock)  // pass mock as the DBTX interface

Setting up expectations:
  CRITICAL — ExpectQuery/ExpectExec use REGEX matching:
    The string passed to ExpectQuery/ExpectExec is compiled as a Go regex.
    Use the EXACT SAME single-line SQL string from the implementation code.
    Define the SQL as a const and share it between implementation and test:

      // In implementation:
      const findByEmailSQL = ` + "`" + `SELECT id, email FROM users WHERE email = $1` + "`" + `
      func (r *Repo) FindByEmail(ctx context.Context, email string) { r.pool.QueryRow(ctx, findByEmailSQL, email) }

      // In test — reference the SAME const:
      mock.ExpectQuery(regexp.QuoteMeta(findByEmailSQL)).WithArgs("alice@example.com").WillReturnRows(rows)

    WRONG patterns that ALWAYS fail:
      mock.ExpectQuery("SELECT id, email \s+FROM users")     ← " \s+" requires 2+ whitespace chars
      mock.ExpectQuery("SELECT id FROM users WHERE id = $1") ← "$1" is regex group ref, not literal
    CORRECT patterns:
      mock.ExpectQuery(regexp.QuoteMeta(findByEmailSQL))      ← best: shares const, escapes regex chars
      mock.ExpectQuery("SELECT id, email FROM users")         ← ok: substring match, no regex metachars

  // For queries returning rows (both Query() and QueryRow() use ExpectQuery):
  rows := pgxmock.NewRows([]string{"id", "name", "email"}).
      AddRow("uuid-1", "Alice", "alice@example.com")
  mock.ExpectQuery(regexp.QuoteMeta(findByEmailSQL)).
      WithArgs("uuid-1").
      WillReturnRows(rows)
  // NOTE: QueryRow() also uses ExpectQuery — there is NO ExpectQueryRow method.

  // For exec (INSERT/UPDATE/DELETE):
  mock.ExpectExec("INSERT INTO users").
      WithArgs("Alice", "alice@example.com").
      WillReturnResult(pgxmock.NewResult("INSERT", 1))

  // Verify all expectations were met:
  if err := mock.ExpectationsWereMet(); err != nil {
      t.Errorf("unmet expectations: %s", err)
  }

Error simulation in tests:
  // For unique constraint violations, use pgconn.PgError (NOT pgx.PgError):
  pgErr := &pgconn.PgError{Code: "23505"}
  mock.ExpectQuery("INSERT").WillReturnError(pgErr)
  // Then check with errors.As: var pgErr *pgconn.PgError; errors.As(err, &pgErr)
  // For "no rows" errors: use pgx.ErrNoRows (from "github.com/jackc/pgx/v5"):
  mock.ExpectQuery("SELECT").WillReturnError(pgx.ErrNoRows)
  // For generic errors: use errors.New("some error")

  ERROR SENTINELS — where they live:
    pgx.ErrNoRows        → "github.com/jackc/pgx/v5"     (CORRECT)
    pgxmock.ErrNoRows    → DOES NOT EXIST (use pgx.ErrNoRows)
    pgxmock.Err*         → pgxmock has NO error sentinels — only pgx and pgconn do

WithArgs matching:
  // Use pgxmock.AnyArg() for arguments you don't care about:
  mock.ExpectExec("INSERT").WithArgs(pgxmock.AnyArg(), "alice@example.com")
`,

	"fiber": `## Fiber v2 API Reference (github.com/gofiber/fiber/v2)

IMPORTANT — these do NOT exist in fiber/v2:
  fiber.As()     ← WRONG, does not exist (use errors.As from stdlib)
  fiber.Is()     ← WRONG, does not exist (use errors.Is from stdlib)

App creation:
  app := fiber.New(fiber.Config{
      ErrorHandler: customErrorHandler,
  })

Route handlers — signature is func(c *fiber.Ctx) error:
  app.Get("/users/:id", getUser)
  app.Post("/users", createUser)
  app.Put("/users/:id", updateUser)
  app.Delete("/users/:id", deleteUser)

Context (c *fiber.Ctx) methods:
  c.Params("id")                    // path parameter
  c.Query("page", "1")             // query parameter with default
  c.BodyParser(&req)               // parse JSON body into struct
  c.Status(201).JSON(data)         // respond with status + JSON body
  c.SendStatus(204)                // respond with status only, no body
  c.Locals("user")                 // get value from middleware context
  c.Locals("user", userObj)        // set value in middleware context

Middleware:
  app.Use(logger.New())
  app.Use(recover.New())
  app.Use(cors.New(cors.Config{AllowOrigins: "http://localhost:3000"}))

Route groups:
  api := app.Group("/api/v1")
  api.Use(authMiddleware)
  api.Get("/users", listUsers)

Error responses:
  return fiber.NewError(fiber.StatusNotFound, "user not found")
  return fiber.NewError(fiber.StatusBadRequest, "invalid input")
  return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})

Testing:
  req := httptest.NewRequest("GET", "/api/v1/users", nil)
  req.Header.Set("Content-Type", "application/json")
  resp, err := app.Test(req, -1)  // -1 = no timeout
`,

	"next": `## Next.js Configuration Rules

CRITICAL: next.config file naming:
- Use next.config.mjs (ESM) — works with ALL Next.js versions including 15.x
- next.config.ts is ONLY supported from Next.js 15.3+; default to .mjs to be safe

npm install vs npm ci in Dockerfiles:
- Use 'npm install' NOT 'npm ci' — package-lock.json is not generated by the pipeline
- npm ci requires an existing package-lock.json and will FAIL without one

Correct Dockerfile pattern for Next.js:
  COPY package*.json ./
  RUN npm install        ← NOT npm ci
  COPY . .
  CMD ["npm", "run", "dev"]
`,

	"fastapi": `## FastAPI API Reference

Router and app setup:
  from fastapi import FastAPI, APIRouter, Depends, HTTPException, status
  app = FastAPI()
  router = APIRouter(prefix="/api/v1", tags=["users"])
  app.include_router(router)

Route definitions:
  @router.get("/users", response_model=list[UserResponse])
  async def list_users(db: AsyncSession = Depends(get_db)) -> list[UserResponse]:
      ...

  @router.post("/users", response_model=UserResponse, status_code=status.HTTP_201_CREATED)
  async def create_user(body: UserCreate, db: AsyncSession = Depends(get_db)) -> UserResponse:
      ...

  @router.get("/users/{user_id}", response_model=UserResponse)
  async def get_user(user_id: UUID, db: AsyncSession = Depends(get_db)) -> UserResponse:
      ...

Pydantic models (v2):
  from pydantic import BaseModel, EmailStr, Field
  class UserCreate(BaseModel):
      name: str = Field(..., min_length=1, max_length=100)
      email: EmailStr

  class UserResponse(BaseModel):
      id: UUID
      name: str
      email: str
      model_config = ConfigDict(from_attributes=True)  # replaces orm_mode=True

Dependency injection:
  async def get_db() -> AsyncGenerator[AsyncSession, None]:
      async with async_session() as session:
          yield session

  async def get_current_user(token: str = Depends(oauth2_scheme)) -> User:
      ...

Error responses:
  raise HTTPException(status_code=404, detail="User not found")
  raise HTTPException(status_code=422, detail=[{"loc": ["body", "email"], "msg": "invalid"}])

CRITICAL: Pydantic v2 breaking changes vs v1:
  orm_mode = True         ← WRONG (v1)
  from_attributes = True  ← CORRECT (v2, inside model_config = ConfigDict(...))
  validator decorator     ← WRONG (v1, use @field_validator in v2)
`,

	"django-drf": `## Django REST Framework API Reference

ViewSets:
  from rest_framework import viewsets, permissions, status
  from rest_framework.response import Response
  from rest_framework.decorators import action

  class UserViewSet(viewsets.ModelViewSet):
      queryset = User.objects.all()
      serializer_class = UserSerializer
      permission_classes = [permissions.IsAuthenticated]

      @action(detail=True, methods=["post"])
      def set_password(self, request, pk=None):
          user = self.get_object()
          ...
          return Response({"status": "password set"})

Serializers:
  from rest_framework import serializers

  class UserSerializer(serializers.ModelSerializer):
      class Meta:
          model = User
          fields = ["id", "name", "email"]
          read_only_fields = ["id"]

  # Nested serializer:
  class OrderSerializer(serializers.ModelSerializer):
      user = UserSerializer(read_only=True)
      class Meta:
          model = Order
          fields = ["id", "user", "total"]

Router:
  from rest_framework.routers import DefaultRouter
  router = DefaultRouter()
  router.register(r"users", UserViewSet)
  urlpatterns = [path("api/", include(router.urls))]

Permissions:
  permission_classes = [permissions.IsAuthenticated]
  permission_classes = [permissions.IsAdminUser]
  permission_classes = [permissions.AllowAny]

Filtering and pagination:
  from rest_framework.filters import SearchFilter, OrderingFilter
  filter_backends = [SearchFilter, OrderingFilter]
  search_fields = ["name", "email"]
`,

	"sqlalchemy": `## SQLAlchemy 2.0 API Reference

CRITICAL: SQLAlchemy 2.0 uses new-style Query API. The 1.x session.query() API still works
but is legacy. Always use the 2.0 select() API.

Model definition (2.0 mapped_column style):
  from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship
  from sqlalchemy import String, Integer, ForeignKey
  import uuid

  class Base(DeclarativeBase):
      pass

  class User(Base):
      __tablename__ = "users"
      id: Mapped[uuid.UUID] = mapped_column(primary_key=True, default=uuid.uuid4)
      name: Mapped[str] = mapped_column(String(100))
      email: Mapped[str] = mapped_column(String(255), unique=True)
      orders: Mapped[list["Order"]] = relationship(back_populates="user")

Async session (preferred for FastAPI/async):
  from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine, async_sessionmaker
  engine = create_async_engine("postgresql+asyncpg://user:pass@host/db")
  async_session = async_sessionmaker(engine, expire_on_commit=False)

Queries (2.0 style — use these, NOT session.query()):
  from sqlalchemy import select, update, delete

  # SELECT
  result = await session.execute(select(User).where(User.email == email))
  user = result.scalar_one_or_none()

  # INSERT
  session.add(User(name="Alice", email="alice@example.com"))
  await session.commit()

  # UPDATE
  await session.execute(update(User).where(User.id == uid).values(name="Bob"))
  await session.commit()

  # DELETE
  await session.execute(delete(User).where(User.id == uid))
  await session.commit()

WRONG (1.x style — do NOT use):
  session.query(User).filter(User.email == email).first()  ← legacy, avoid
`,

	"golang-jwt": `## golang-jwt/v5 API Reference (github.com/golang-jwt/jwt/v5)

Creating a token:
  claims := jwt.MapClaims{
      "sub": userID,
      "exp": jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
      "iat": jwt.NewNumericDate(time.Now()),
  }
  token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
  signedString, err := token.SignedString([]byte(secretKey))

Parsing a token:
  token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
      if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
          return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
      }
      return []byte(secretKey), nil
  })
  if err != nil || !token.Valid {
      return fmt.Errorf("invalid token")
  }
  claims, ok := token.Claims.(jwt.MapClaims)
`,

	"godotenv": `## godotenv API Reference (github.com/joho/godotenv)

CRITICAL — Go does NOT auto-load .env files:
  os.Getenv("DATABASE_URL") only reads the OS environment. If your config is in a .env file,
  you MUST call godotenv.Load() BEFORE any os.Getenv() call, otherwise the app crashes on startup
  with "required environment variable not set" even when the .env file exists.

Usage in main.go (MUST be the first thing in main or run):
  import "github.com/joho/godotenv"

  func main() {
      // Load .env file. Ignore error — .env is optional in production where
      // env vars are set by the orchestrator (Docker, k8s, systemd, etc.).
      _ = godotenv.Load()

      // Now os.Getenv works for values defined in .env
      dsn := os.Getenv("DATABASE_URL")
  }

DO NOT use godotenv.Overload() — it overwrites env vars already set by the OS,
which breaks Docker/k8s deployments that pass env vars via the runtime.
`,
}
