---
name: file-storage-patterns
description: File storage implementation patterns — presigned URLs for S3/GCS/MinIO, streaming upload/download, MIME validation, size enforcement, virus scanning, and TUS resumable uploads.
origin: vibemenu
---

# File Storage Patterns

Never proxy large files through the application server. Use presigned URLs to let clients communicate directly with object storage. This skill covers the full upload/download lifecycle with correct streaming, validation, and security.

## When to Activate

- File upload/download in any backend service
- Implementing user avatar, document, or media upload flows
- Streaming large files without loading them into memory
- Integrating with S3, GCS, Azure Blob, MinIO, or R2

## Presigned URL Flow (Preferred Architecture)

```
Client                      App Server                  Object Storage (S3/GCS/MinIO)
  |                              |                               |
  |-- POST /files/upload-url --> |                               |
  |   { filename, content_type } |                               |
  |                              |-- GeneratePresignedPUT ------> |
  |                              |<-- presigned_url ------------- |
  |<-- { upload_url, file_key } -|                               |
  |                              |                               |
  |-- PUT presigned_url -------> (direct upload, no app server) ->|
  |<-- 200 OK (from storage) ------------------------------------ |
  |                              |                               |
  |-- POST /files/confirm -----> |                               |
  |   { file_key }               |-- HeadObject (verify) ------> |
  |                              |<-- metadata ------------------ |
  |<-- { file_id } ------------- |                               |
```

## Presigned URL Generation

### AWS S3 (Go — `aws-sdk-go-v2`)

```go
import (
    "context"
    "fmt"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    smithytime "github.com/aws/smithy-go/time"
)

type S3Uploader struct {
    client    *s3.Client
    presigner *s3.PresignClient
    bucket    string
}

func NewS3Uploader(ctx context.Context, bucket string) (*S3Uploader, error) {
    cfg, err := config.LoadDefaultConfig(ctx)
    if err != nil {
        return nil, fmt.Errorf("load aws config: %w", err)
    }
    client := s3.NewFromConfig(cfg)
    return &S3Uploader{
        client:    client,
        presigner: s3.NewPresignClient(client),
        bucket:    bucket,
    }, nil
}

// GenerateUploadURL creates a presigned PUT URL valid for 15 minutes.
// Always constrain ContentType — prevents content-type spoofing.
func (u *S3Uploader) GenerateUploadURL(
    ctx context.Context,
    key string,
    contentType string,
    maxBytes int64,
) (string, error) {
    req, err := u.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
        Bucket:         aws.String(u.bucket),
        Key:            aws.String(key),
        ContentType:    aws.String(contentType),
        ContentLength:  aws.Int64(maxBytes),
    }, func(opts *s3.PresignOptions) {
        opts.Expires = 15 * time.Minute
    })
    if err != nil {
        return "", fmt.Errorf("presign put object: %w", err)
    }
    return req.URL, nil
}

// GenerateDownloadURL creates a presigned GET URL valid for 1 hour.
func (u *S3Uploader) GenerateDownloadURL(ctx context.Context, key string) (string, error) {
    req, err := u.presigner.PresignGetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(u.bucket),
        Key:    aws.String(key),
    }, func(opts *s3.PresignOptions) {
        opts.Expires = time.Hour
    })
    if err != nil {
        return "", fmt.Errorf("presign get object: %w", err)
    }
    return req.URL, nil
}
```

### AWS S3 (Python — boto3)

```python
import boto3
from botocore.exceptions import ClientError

s3_client = boto3.client("s3", region_name="us-east-1")

def generate_upload_url(bucket: str, key: str, content_type: str, expires_in: int = 900) -> str:
    """Generate presigned URL for client-side PUT. expires_in in seconds (default 15 min)."""
    return s3_client.generate_presigned_url(
        "put_object",
        Params={
            "Bucket": bucket,
            "Key": key,
            "ContentType": content_type,  # constrain allowed content type
        },
        ExpiresIn=expires_in,
        HttpMethod="PUT",
    )

def generate_download_url(bucket: str, key: str, expires_in: int = 3600) -> str:
    return s3_client.generate_presigned_url(
        "get_object",
        Params={"Bucket": bucket, "Key": key},
        ExpiresIn=expires_in,
    )
```

### GCS (Go — `cloud.google.com/go/storage`)

```go
import (
    "cloud.google.com/go/storage"
    "google.golang.org/api/option"
)

func GenerateGCSUploadURL(
    ctx context.Context,
    bucket, object, contentType string,
) (string, error) {
    client, err := storage.NewClient(ctx, option.WithoutAuthentication())
    if err != nil {
        return "", err
    }
    defer client.Close()

    opts := &storage.SignedURLOptions{
        Scheme:      storage.SigningSchemeV4,
        Method:      "PUT",
        ContentType: contentType,
        Expires:     time.Now().Add(15 * time.Minute),
    }
    return client.Bucket(bucket).SignedURL(object, opts)
}
```

### MinIO (Go — S3-Compatible)

```go
import "github.com/minio/minio-go/v7"

func GenerateMinIOUploadURL(
    ctx context.Context,
    client *minio.Client,
    bucket, key string,
    expires time.Duration,
) (string, error) {
    u, err := client.PresignedPutObject(ctx, bucket, key, expires)
    if err != nil {
        return "", fmt.Errorf("minio presign: %w", err)
    }
    return u.String(), nil
}
```

## Server-Side Multipart Upload Handler (When Presigned URLs Can't Be Used)

```go
const maxUploadBytes = 50 << 20 // 50 MB

func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
    // Enforce size limit at the stream level — before any parsing
    r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

    if err := r.ParseMultipartForm(10 << 20); err != nil {
        if errors.Is(err, http.ErrHandlerTimeout) || strings.Contains(err.Error(), "request body too large") {
            http.Error(w, "File too large (max 50MB)", http.StatusRequestEntityTooLarge)
            return
        }
        http.Error(w, "Failed to parse upload", http.StatusBadRequest)
        return
    }
    defer r.MultipartForm.RemoveAll()

    f, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, "Missing file field", http.StatusBadRequest)
        return
    }
    defer f.Close()

    // Validate MIME type before writing to storage
    if err := validateMIMEType(f, allowedMIMETypes); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    key := generateStorageKey(header.Filename)

    // Stream directly to storage — never io.ReadAll for large files
    if err := h.storage.Upload(r.Context(), key, f, header.Size); err != nil {
        h.log.Error("upload failed", "key", key, "error", err)
        http.Error(w, "Upload failed", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"file_id": key})
}
```

## MIME Type Validation (Check Magic Bytes)

```go
var allowedMIMETypes = map[string]bool{
    "image/jpeg": true,
    "image/png":  true,
    "image/webp": true,
    "image/gif":  true,
    "application/pdf": true,
}

// validateMIMEType reads the first 512 bytes and uses Go's content detection.
// The reader is seeked back to the start after reading.
func validateMIMEType(r io.ReadSeeker, allowed map[string]bool) error {
    buf := make([]byte, 512)
    n, err := r.Read(buf)
    if err != nil && !errors.Is(err, io.EOF) {
        return fmt.Errorf("read file header: %w", err)
    }
    if _, err := r.Seek(0, io.SeekStart); err != nil {
        return fmt.Errorf("seek after mime check: %w", err)
    }

    detected := http.DetectContentType(buf[:n])
    // DetectContentType returns "image/jpeg; charset=..." — strip parameters
    mimeType := strings.Split(detected, ";")[0]

    if !allowed[mimeType] {
        return fmt.Errorf("file type %q not allowed", mimeType)
    }
    return nil
}
```

```python
# Python — python-magic (install: pip install python-magic)
import magic

ALLOWED_TYPES = {"image/jpeg", "image/png", "image/webp", "application/pdf"}

def validate_mime_type(file_bytes: bytes) -> str:
    """Returns the detected MIME type or raises ValueError."""
    detected = magic.from_buffer(file_bytes[:1024], mime=True)
    if detected not in ALLOWED_TYPES:
        raise ValueError(f"File type '{detected}' is not permitted")
    return detected
```

```typescript
// Node.js — file-type package (npm install file-type)
import { fileTypeFromBuffer } from 'file-type';

const ALLOWED_TYPES = new Set(['image/jpeg', 'image/png', 'image/webp', 'application/pdf']);

async function validateMimeType(buffer: Buffer): Promise<string> {
  const result = await fileTypeFromBuffer(buffer.slice(0, 1024));
  if (!result || !ALLOWED_TYPES.has(result.mime)) {
    throw new Error(`File type '${result?.mime ?? 'unknown'}' is not permitted`);
  }
  return result.mime;
}

// ❌ NEVER trust the Content-Type header:
// const contentType = req.headers['content-type']; // client-controlled — can lie
```

## Streaming Download

```go
// Go — stream from S3 to HTTP response without buffering in memory
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
    key := r.PathValue("key") // Go 1.22+ path params

    obj, err := h.s3.GetObject(r.Context(), &s3.GetObjectInput{
        Bucket: aws.String(h.bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        http.Error(w, "File not found", http.StatusNotFound)
        return
    }
    defer obj.Body.Close()

    w.Header().Set("Content-Type", aws.ToString(obj.ContentType))
    w.Header().Set("Content-Length", fmt.Sprintf("%d", aws.ToInt64(obj.ContentLength)))
    w.Header().Set("Content-Disposition",
        fmt.Sprintf(`attachment; filename="%s"`, sanitizeFilename(key)))

    // Stream — zero memory overhead for large files
    if _, err := io.Copy(w, obj.Body); err != nil {
        h.log.Error("stream download failed", "key", key, "error", err)
    }
}
```

## Virus Scanning Hook

Define a `VirusScanner` interface so the no-op and real implementations are swappable:

```go
// internal/storage/scanner.go

type VirusScanner interface {
    Scan(ctx context.Context, r io.Reader) error
}

// NoOpScanner — default; use in development or when ClamAV is unavailable
type NoOpScanner struct{}

func (n *NoOpScanner) Scan(_ context.Context, _ io.Reader) error {
    return nil
}

// ClamAVScanner — production
type ClamAVScanner struct {
    addr string // e.g. "localhost:3310"
}

func (c *ClamAVScanner) Scan(ctx context.Context, r io.Reader) error {
    conn, err := net.DialTimeout("tcp", c.addr, 5*time.Second)
    if err != nil {
        return fmt.Errorf("clamd connect: %w", err)
    }
    defer conn.Close()

    // Send INSTREAM command and pipe file data
    fmt.Fprintf(conn, "zINSTREAM\x00")
    buf := make([]byte, 4096)
    for {
        n, readErr := r.Read(buf)
        if n > 0 {
            // Each chunk prefixed with 4-byte big-endian length
            binary.Write(conn, binary.BigEndian, uint32(n))
            conn.Write(buf[:n])
        }
        if errors.Is(readErr, io.EOF) {
            break
        }
        if readErr != nil {
            return fmt.Errorf("read file for scan: %w", readErr)
        }
    }
    binary.Write(conn, binary.BigEndian, uint32(0)) // zero-length chunk = end

    response, _ := bufio.NewReader(conn).ReadString('\n')
    if strings.Contains(response, "FOUND") {
        return fmt.Errorf("virus detected: %s", strings.TrimSpace(response))
    }
    return nil
}

// Usage in upload handler (scan before finalizing storage)
func (h *FileHandler) uploadWithScan(ctx context.Context, key string, f io.ReadSeeker) error {
    if err := h.scanner.Scan(ctx, f); err != nil {
        return fmt.Errorf("virus scan: %w", err)
    }
    if _, err := f.Seek(0, io.SeekStart); err != nil {
        return err
    }
    return h.storage.Upload(ctx, key, f)
}
```

## TUS Resumable Uploads (Files > 100MB)

TUS is an open protocol for resumable file uploads. Use it when reliability for large files matters.

```go
// Go server — tusd library
import (
    "github.com/tus/tusd/v2/pkg/filelocker"
    "github.com/tus/tusd/v2/pkg/handler"
    "github.com/tus/tusd/v2/pkg/s3store"
)

func SetupTUSHandler(s3Client *s3.Client, bucket string) (http.Handler, error) {
    store := s3store.New(bucket, s3Client)
    locker := filelocker.New("/tmp/tusd-locks")

    composer := handler.NewStoreComposer()
    store.UseIn(composer)
    locker.UseIn(composer)

    h, err := handler.NewHandler(handler.Config{
        BasePath:              "/files/",
        StoreComposer:         composer,
        MaxSize:               5 * 1024 * 1024 * 1024, // 5 GB max
        NotifyCompleteUploads: true,
    })
    if err != nil {
        return nil, err
    }

    // React to completed uploads
    go func() {
        for event := range h.CompleteUploads {
            log.Printf("Upload complete: %s (%d bytes)", event.Upload.ID, event.Upload.Size)
            // trigger post-processing job here
        }
    }()

    return http.StripPrefix("/files/", h), nil
}
```

```typescript
// Client — tus-js-client (npm install tus-js-client)
import * as tus from 'tus-js-client';

function uploadFile(file: File, onProgress: (pct: number) => void): Promise<string> {
  return new Promise((resolve, reject) => {
    const upload = new tus.Upload(file, {
      endpoint: '/files/',
      retryDelays: [0, 3000, 5000, 10000, 20000],
      metadata: {
        filename: file.name,
        filetype: file.type,
      },
      onError: reject,
      onProgress(bytesUploaded, bytesTotal) {
        onProgress(Math.round((bytesUploaded / bytesTotal) * 100));
      },
      onSuccess() {
        resolve(upload.url!);
      },
    });

    // Resume an existing upload if url stored from previous session
    upload.findPreviousUploads().then((previous) => {
      if (previous.length > 0) {
        upload.resumeFromPreviousUpload(previous[0]);
      }
      upload.start();
    });
  });
}
```

## Anti-Patterns

```go
// ❌ BAD: Loading entire file into memory
data, err := io.ReadAll(r.Body)  // crashes on 1GB uploads

// ✅ GOOD: Stream with io.Copy
io.Copy(storageWriter, r.Body)

// ❌ BAD: Trusting client Content-Type header for MIME validation
contentType := r.Header.Get("Content-Type")  // attacker-controlled

// ✅ GOOD: Detect from file magic bytes
validateMIMEType(file)

// ❌ BAD: Serving files by proxying through app server (memory/CPU waste)
data, _ := s3.GetObject(...)
io.ReadAll(data.Body)  // buffer full file
w.Write(data)

// ✅ GOOD: Presigned URL for direct client-to-storage download
downloadURL, _ := GenerateDownloadURL(ctx, key)
http.Redirect(w, r, downloadURL, http.StatusTemporaryRedirect)

// ❌ BAD: No size limit on upload — DoS risk
r.ParseMultipartForm(32 << 20)  // without MaxBytesReader

// ✅ GOOD: Enforce size at stream level before parsing
r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
r.ParseMultipartForm(10 << 20)
```
