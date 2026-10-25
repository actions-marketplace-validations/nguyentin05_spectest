# SpecTest Web Control Center: API Documentation
> **Phiên bản:** v2.0 (Chuẩn hóa theo Kiến trúc 2 File: `spectest.yaml` + `spectest.log.yaml`)
> **Mặc định phục vụ tại:** `http://localhost:4173/api` (khi chạy `spectest ui`)

---

## MỤC LỤC
1. [Quy chuẩn chung (General Conventions)](#1-quy-chuẩn-chung-general-conventions)
2. [Hệ thống & Trạng thái (System & Health)](#2-hệ-thống--trạng-thái-system--health)
3. [Cấu hình AI Model (LLM Configuration)](#3-cấu-hình-ai-model-llm-configuration)
4. [Kiểm tra máy chủ mục tiêu (Target Server Probing)](#4-kiểm-tra-máy-chủ-mục-tiêu-target-server-probing)
5. [Quản lý Workspace & Tập tin (Workspace & File Specs)](#5-quản-lý-workspace--tập-tin-workspace--file-specs)
6. [Đặc tả Kịch bản & Lịch sử (Suite & Audit History)](#6-đặc-tả-kịch-bản--lịch-sử-suite--audit-history)
7. [Khởi tạo Kịch bản Kiểm thử (Test Suite Generation)](#7-khởi-tạo-kịch-bản-kiểm-thử-test-suite-generation)
8. [Thực thi & SSE Realtime Streaming (Runner & Telemetry)](#8-thực-thi--sse-realtime-streaming-runner--telemetry)
9. [Đọc tài liệu Markdown (Markdown Reader)](#9-đọc-tài-liệu-markdown-markdown-reader)

---

## 1. QUY CHUẨN CHUNG (GENERAL CONVENTIONS)

### 1.1. Base URL & Protocol
* Tất cả các API nghiệp vụ đều có tiền tố `/api`.
* Giao tiếp qua giao thức **HTTP/1.1** hoặc **HTTP/2**, định dạng dữ liệu mặc định là `application/json; charset=utf-8`.
* Luồng tiến độ thời gian thực (Real-time progress) sử dụng chuẩn **Server-Sent Events (SSE)** tại endpoint `/api/run/stream`.
* Server tự động bật **CORS** (`Access-Control-Allow-Origin: *`, `Allow-Methods`, `Allow-Headers`) hỗ trợ các dev server frontend như Vite/Webpack chạy ở port khác (ví dụ: `http://localhost:5173`).

### 1.2. Định dạng phản hồi chuẩn (Response Envelopes)
* **Thành công (Success):** Trả về HTTP Status `200 OK` (hoặc `201 Created`, `202 Accepted`) kèm payload JSON.
* **Lỗi (Error):** Trả về HTTP Status `4xx` hoặc `5xx` với cấu trúc thống nhất:
```json
{
  "success": false,
  "error": "Mô tả chi tiết nguyên nhân lỗi"
}
```

---

## 2. HỆ THỐNG & TRẠNG THÁI (SYSTEM & HEALTH)

### 2.1. `GET /api/health`
Kiểm tra tính sẵn sàng của UI server và thông tin thư mục làm việc hiện tại.

* **Method:** `GET`
* **Response `200 OK`:**
```json
{
  "status": "ok",
  "time": "2026-10-25T13:45:00+07:00",
  "work_dir": "/home/tin/Project/current/spectest"
}
```

### 2.2. `GET /api/logs`
Lấy danh sách log hệ thống được ghi nhận trong bộ nhớ của UI session manager.

* **Method:** `GET`
* **Response `200 OK`:**
```json
{
  "logs": [
    "[13:40:01] SpecTest Web UI Server listening on :4173",
    "[13:41:05] Uploaded file: auth-service.yaml",
    "[13:42:10] Test run scheduled against http://localhost:8080"
  ]
}
```

---

## 3. CẤU HÌNH AI MODEL (LLM CONFIGURATION)

Quản lý thông tin nhà cung cấp AI (Google Gemini, OpenAI, hoặc Local Model Server như Ollama/vLLM/LocalAI).

### 3.1. `GET /api/config`
Lấy cấu hình LLM đang hoạt động. Khóa API (`api_key`) luôn được ẩn giấu (masked) vì lý do bảo mật.

* **Method:** `GET`
* **Response `200 OK`:**
```json
{
  "provider": "gemini",
  "model": "gemini-1.5-flash",
  "endpoint": "https://generativelanguage.googleapis.com",
  "has_api_key": true,
  "api_key_masked": "AIzaSy...4xK9"
}
```

### 3.2. `POST /api/config`
Cập nhật thông tin provider, endpoint, model hoặc API key.

* **Method:** `POST`
* **Request Headers:** `Content-Type: application/json`
* **Request Body:**
```json
{
  "provider": "openai",
  "endpoint": "https://api.openai.com/v1",
  "model": "gpt-4o-mini",
  "api_key": "sk-proj-xxxxxxxxxxxxxxxx"
}
```
> **Ghi chú:**
> * Đối với Local Server (`provider: "local"` hoặc endpoint nội bộ dạng `http://localhost:...`, `http://127.0.0.1:...`), trường `api_key` là tùy chọn (optional).
> * Đối với Cloud Provider (`gemini`, `openai`), bắt buộc phải có `api_key`.

* **Response `200 OK`:**
```json
{
  "success": true,
  "message": "Configuration updated successfully"
}
```

### 3.3. `POST /api/config/test`
Kiểm tra kết nối tới LLM Provider bằng cách gửi prompt thử nghiệm ngắn và đo thời gian phản hồi.

* **Method:** `POST`
* **Request Body:**
```json
{
  "provider": "gemini",
  "endpoint": "https://generativelanguage.googleapis.com",
  "model": "gemini-1.5-flash",
  "api_key": "AIzaSy..."
}
```
* **Response `200 OK`:**
```json
{
  "success": true,
  "latency_ms": 482,
  "provider": "gemini",
  "model": "gemini-1.5-flash",
  "message": "Connection to Gemini succeeded"
}
```
* **Response `400 Bad Request`:**
```json
{
  "success": false,
  "latency_ms": 210,
  "error": "LLM ping failed: 401 Unauthorized - invalid API key"
}
```

---

## 4. KIỂM TRA MÁY CHỦ MỤC TIÊU (TARGET SERVER PROBING)

### 4.1. `POST /api/server/healthcheck`
Thực hiện pre-flight healthcheck tới target base URL trước khi chạy test, kiểm tra xem API backend đã boot và sẵn sàng tiếp nhận request hay chưa.

* **Method:** `POST`
* **Request Body:**
```json
{
  "target_url": "http://localhost:8080",
  "timeout_seconds": 5
}
```
* **Response `200 OK`:**
```json
{
  "ready": true,
  "target_url": "http://localhost:8080",
  "latency_ms": 15,
  "message": "Target server is healthy and responding"
}
```

---

## 5. QUẢN LÝ WORKSPACE & TẬP TIN (WORKSPACE & FILE SPECS)

### 5.1. `GET /api/specs/workspace`
Quét toàn bộ thư mục làm việc của dự án và phân loại các tập tin phục vụ kiểm thử.

* **Method:** `GET`
* **Response `200 OK`:**
```json
{
  "current_dir": "/home/tin/Project/current/spectest",
  "items": [
    {
      "name": "specs/auth-api.yaml",
      "path": "/home/tin/Project/current/spectest/specs/auth-api.yaml",
      "type": "openapi",
      "size": 18450
    },
    {
      "name": "specs/order-api.json",
      "path": "/home/tin/Project/current/spectest/specs/order-api.json",
      "type": "openapi",
      "size": 24102
    },
    {
      "name": "docs/requirements.md",
      "path": "/home/tin/Project/current/spectest/docs/requirements.md",
      "type": "prd",
      "size": 6512
    },
    {
      "name": "tests/spectest.yaml",
      "path": "/home/tin/Project/current/spectest/tests/spectest.yaml",
      "type": "suite",
      "size": 8940
    },
    {
      "name": "tests/spectest.log.yaml",
      "path": "/home/tin/Project/current/spectest/tests/spectest.log.yaml",
      "type": "history",
      "size": 14210
    }
  ]
}
```
> **Phân loại `type`:** `openapi` (Spec API), `prd` (Requirements markdown), `suite` (`spectest.yaml`), `history` (`spectest.log.yaml`), `matrix` (Báo cáo markdown), `other`.

### 5.2. `POST /api/specs/upload`
Tải lên tập tin đặc tả (OpenAPI, PRD, hoặc kịch bản). Hỗ trợ cả `multipart/form-data` lẫn JSON raw text.

* **Cách 1: Multipart File Upload**
  * `Content-Type: multipart/form-data`
  * Form Field: `file`

* **Cách 2: Raw Text JSON**
  * `Content-Type: application/json`
  * Body:
    ```json
    {
      "file_name": "openapi-petstore.yaml",
      "content": "openapi: 3.0.0\ninfo:\n  title: Petstore..."
    }
    ```

* **Response `200 OK`:**
```json
{
  "success": true,
  "file_name": "openapi-petstore.yaml",
  "path": "/home/tin/Project/current/spectest/.spectest_ui_uploads/openapi-petstore.yaml",
  "type": "openapi",
  "size": 4210
}
```

---

## 6. ĐẶC TẢ KỊCH BẢN & LỊCH SỬ (SUITE & AUDIT HISTORY)

Đây là 2 endpoint trọng tâm phản ánh trực tiếp **Mô hình 2 File chuẩn** (`spectest.yaml` và `spectest.log.yaml`).

### 6.1. `GET /api/specs/plan` (Current Suite State)
Đọc và phân tích cú pháp file `spectest.yaml` v1 hiện tại, trả về toàn bộ dữ liệu có cấu trúc để UI dựng cây Scenario, danh sách Setup và các bước HTTP.

* **Method:** `GET`
* **Query Parameters:**
  * `path` *(optional)*: Đường dẫn tới file `spectest.yaml` (nếu để trống, tự động tìm trong workspace hoặc thư mục `./tests/spectest.yaml`).
* **Response `200 OK`:**
```json
{
  "path": "tests/spectest.yaml",
  "version": 1,
  "servers": ["auth", "order"],
  "apis": [
    "POST {auth}/api/register",
    "POST {auth}/api/login",
    "POST {order}/api/products",
    "POST {order}/api/orders"
  ],
  "analysis": "- Có register + login nên tạo được user mới độc lập cho từng scenario.\n- Requirements có sẵn tài khoản admin dùng tạo product.",
  "seed": {
    "admin": {
      "vars": {
        "admin_email": "admin@shop.com",
        "admin_password": "Admin@123"
      }
    }
  },
  "setup": {
    "admin_session": {
      "vars": {
        "email": "{{admin_email}}",
        "password": "{{admin_password}}"
      },
      "steps": [
        {
          "call": "POST {auth}/api/login",
          "headers": {},
          "body": {
            "email": "{{email}}",
            "password": "{{password}}"
          },
          "expect": 200,
          "save": {
            "admin_token": "body.access_token"
          }
        }
      ]
    },
    "user": {
      "vars": {
        "email": "user_{{random}}@test.com",
        "password": "P@ssw0rd123"
      },
      "steps": [
        {
          "call": "POST {auth}/api/register",
          "body": {
            "email": "{{email}}",
            "password": "{{password}}"
          },
          "expect": 201,
          "save": {
            "user_id": "body.id"
          }
        }
      ]
    },
    "logged_in_user": {
      "use": ["user"],
      "steps": [
        {
          "call": "POST {auth}/api/login",
          "body": {
            "email": "{{email}}",
            "password": "{{password}}"
          },
          "expect": 200,
          "save": {
            "user_token": "body.access_token"
          }
        }
      ]
    }
  },
  "scenarios": [
    {
      "id": "login-wrong-password",
      "name": "Login sai password",
      "setup": ["user"],
      "steps": [
        {
          "call": "POST {auth}/api/login",
          "body": {
            "email": "{{email}}",
            "password": "sai_mat_khau"
          },
          "expect": 401
        }
      ],
      "locked": false,
      "skip": ""
    },
    {
      "id": "order-over-stock",
      "name": "Đặt hàng vượt tồn kho",
      "setup": ["logged_in_user"],
      "steps": [
        {
          "call": "POST {order}/api/orders",
          "headers": {
            "Authorization": "Bearer {{user_token}}"
          },
          "body": {
            "items": [{"product_id": "prod_1", "quantity": 999}]
          },
          "expect": 422
        }
      ],
      "locked": true,
      "skip": ""
    }
  ],
  "stats": {
    "total_scenarios": 2,
    "total_setups": 3,
    "total_steps": 4,
    "locked_scenarios": 1,
    "skipped_scenarios": 0
  }
}
```

### 6.2. `GET /api/specs/history` (Audit History Log)
Đọc file `spectest.log.yaml` và trả về danh sách các sự kiện tiến hóa qua thời gian (Event Sourcing).

* **Method:** `GET`
* **Query Parameters:**
  * `dir` *(optional)*: Thư mục chứa `spectest.log.yaml` (mặc định `./tests`).
* **Response `200 OK`:**
```json
{
  "base": {
    "at": "2026-10-20T09:00:00+07:00",
    "by": "ai",
    "after_hash": "5c1d02a9b7e4",
    "spectest_summary": {
      "servers": ["auth", "order"],
      "apis_count": 4,
      "scenarios_count": 2
    }
  },
  "updates": [
    {
      "n": 1,
      "at": "2026-10-24T10:15:00+07:00",
      "by": "ai",
      "note": "API added: POST {order}/api/orders/{id}/cancel",
      "trigger": {
        "openapi": {
          "added": ["POST {order}/api/orders/{id}/cancel"],
          "changed": ["POST {auth}/api/register"],
          "removed": []
        },
        "requirements": {
          "before_hash": "a1b2c3d4e5f6",
          "after_hash": "f6e5d4c3b2a1"
        }
      },
      "create": {
        "setup_added": ["order"],
        "scenarios_added": ["order-cancel-pending"]
      },
      "change": [
        {
          "target": "setup",
          "id": "user",
          "reason": "register bổ sung field phone bắt buộc",
          "before": {"vars": {"email": "user_{{random}}@test.com"}},
          "after": {"vars": {"email": "user_{{random}}@test.com", "phone": "0912345678"}}
        }
      ],
      "after_hash": "8e41ab77c9d0"
    }
  ]
}
```

---

## 7. KHỞI TẠO KỊCH BẢN KIỂM THỬ (TEST SUITE GENERATION)

### 7.1. `POST /api/generate`
Kích hoạt pipeline sinh kịch bản từ một hoặc nhiều file OpenAPI và file PRD requirements thô.

* **Method:** `POST`
* **Request Body:**
```json
{
  "apis": [
    {
      "server": "auth",
      "path": "specs/auth-api.yaml"
    },
    {
      "server": "order",
      "path": "specs/order-api.yaml"
    }
  ],
  "req_path": "docs/requirements.md",
  "output_dir": "./tests",
  "dry_run": false
}
```
* **Response `200 OK` (Thành công):**
```json
{
  "success": true,
  "suite_path": "./tests/spectest.yaml",
  "log_path": "./tests/spectest.log.yaml",
  "servers": ["auth", "order"],
  "apis_count": 8,
  "scenarios_generated": 6,
  "message": "Test suite generated and saved successfully"
}
```
* **Response `202 Accepted` (Đang được Core team hoàn thiện pipeline LLM):**
```json
{
  "success": true,
  "status": "pending_synthesis",
  "message": "Inputs verified and normalized. Core synthesis engine is processing."
}
```

---

## 8. THỰC THI & SSE REALTIME STREAMING (RUNNER & TELEMETRY)

### 8.1. `POST /api/run`
Khởi động phiên chạy kiểm thử ngầm (Background Test Run Session).

* **Method:** `POST`
* **Request Body:**
```json
{
  "suite_path": "tests/spectest.yaml",
  "target_url": "http://localhost:8080",
  "targets": {
    "auth": "http://localhost:8001",
    "order": "http://localhost:8002"
  },
  "timeout_seconds": 30
}
```
> **Lưu ý cấu hình URL:**
> * Nếu truyền `targets`: runner ánh xạ từng server alias (`{auth}`, `{order}`) tới base URL riêng biệt.
> * Nếu truyền `target_url`: runner dùng chung base URL này cho mọi server trong suite.

* **Response `202 Accepted`:**
```json
{
  "run_id": "run-a1b2c3d4",
  "status": "running",
  "message": "Test execution started",
  "target_url": "http://localhost:8080",
  "suite_path": "tests/spectest.yaml"
}
```

### 8.2. `GET /api/run/stream` (SSE Realtime Stream)
Kênh Server-Sent Events phát tiến độ thực thi từng bước HTTP cho UI cập nhật live dashboard.

* **Method:** `GET`
* **Response Headers:**
  * `Content-Type: text/event-stream`
  * `Cache-Control: no-cache`
  * `Connection: keep-alive`

#### Các loại sự kiện (`event.type`):

1. **`start`** – Bắt đầu chạy test suite:
```json
data: {
  "type": "start",
  "timestamp": "2026-10-25T13:46:00Z",
  "message": "Beginning execution of suite",
  "data": {
    "suite_path": "tests/spectest.yaml",
    "total_scenarios": 2,
    "servers": ["auth", "order"]
  }
}
```

2. **`scenario_start`** – Bắt đầu thực thi một kịch bản:
```json
data: {
  "type": "scenario_start",
  "timestamp": "2026-10-25T13:46:01Z",
  "message": "Executing scenario: order-over-stock",
  "data": {
    "scenario_id": "order-over-stock",
    "scenario_name": "Đặt hàng vượt tồn kho",
    "setups_to_run": ["user", "logged_in_user"]
  }
}
```

3. **`step_result`** – Kết quả của từng bước HTTP đơn nhất:
```json
data: {
  "type": "step_result",
  "timestamp": "2026-10-25T13:46:02Z",
  "data": {
    "scenario_id": "order-over-stock",
    "call": "POST {auth}/api/login",
    "method": "POST",
    "url": "http://localhost:8001/api/login",
    "status_code": 200,
    "expected_code": 200,
    "latency_ms": 32,
    "passed": true,
    "saved_vars": {
      "user_token": "eyJhbGciOi..."
    }
  }
}
```

4. **`scenario_complete`** – Hoàn tất một kịch bản:
```json
data: {
  "type": "scenario_complete",
  "timestamp": "2026-10-25T13:46:03Z",
  "data": {
    "scenario_id": "order-over-stock",
    "passed": true,
    "duration_ms": 115
  }
}
```

5. **`complete`** – Kết thúc toàn bộ bài test:
```json
data: {
  "type": "complete",
  "timestamp": "2026-10-25T13:46:04Z",
  "message": "All test scenarios completed",
  "data": {
    "total_scenarios": 2,
    "passed_scenarios": 2,
    "failed_scenarios": 0,
    "skipped_scenarios": 0,
    "total_checks": 5,
    "duration_ms": 254
  }
}
```

6. **`error` / `cancelled`** – Thất bại do lỗi hệ thống hoặc do người dùng hủy:
```json
data: {
  "type": "error",
  "timestamp": "2026-10-25T13:46:05Z",
  "message": "Pre-flight healthcheck failed: connection refused to http://localhost:8080"
}
```

### 8.3. `POST /api/run/cancel`
Hủy phiên test đang chạy dở.

* **Method:** `POST`
* **Response `200 OK`:**
```json
{
  "success": true,
  "message": "Test run cancelled successfully"
}
```

### 8.4. `GET /api/run/report`
Lấy toàn bộ báo cáo telemetry chi tiết của phiên test gần nhất.

* **Method:** `GET`
* **Response `200 OK`:**
```json
{
  "suite_path": "tests/spectest.yaml",
  "status": "passed",
  "start_time": "2026-10-25T13:46:00Z",
  "end_time": "2026-10-25T13:46:04Z",
  "duration_ms": 254,
  "total_scenarios": 2,
  "passed_scenarios": 2,
  "failed_scenarios": 0,
  "skipped_scenarios": 0,
  "records": [
    {
      "scenario_id": "order-over-stock",
      "scenario_name": "Đặt hàng vượt tồn kho",
      "step_index": 1,
      "call": "POST {order}/api/orders",
      "method": "POST",
      "url": "http://localhost:8002/api/orders",
      "passed": true,
      "status_code": 422,
      "expected_code": 422,
      "latency_ms": 45,
      "fail_reason": "",
      "ai_diagnosis": "",
      "request_payload": "{\"items\":[{\"product_id\":\"prod_1\",\"quantity\":999}]}",
      "response_body": "{\"error\":\"insufficient stock\"}"
    }
  ]
}
```

---

## 9. ĐỌC TÀI LIỆU MARKDOWN (MARKDOWN READER)

### 9.1. `GET /api/markdown/read`
Đọc nội dung thô của các tài liệu markdown (PRD, Requirements, Coverage Matrix) để hiển thị trên trình xem Markdown của UI.

* **Method:** `GET`
* **Query Parameters:**
  * `path` *(required)*: Đường dẫn tới file `.md`.
* **Response `200 OK`:**
```json
{
  "path": "docs/requirements.md",
  "file_name": "requirements.md",
  "content": "# Product Requirements Document\n\n## 1. Authentication\nUser must register..."
}
```
* **Response `404 Not Found`:**
```json
{
  "success": false,
  "error": "File not found: docs/missing.md"
}
```

---

## 10. HƯỚNG DẪN TÍCH HỢP

1. **Khởi chạy UI Backend Server:**
   ```bash
   spectest ui --port 4173 --work-dir .
   ```
2. **Khởi chạy Frontend Dev Server (Vite / React / Vue):**
   * Cấu hình API Proxy trong `vite.config.js`:
     ```js
     server: {
       proxy: {
         '/api': 'http://localhost:4173'
       }
     }
     ```
   * Hoặc gọi trực tiếp với Base URL `http://localhost:4173`.
3. **Lắng nghe SSE trong Javascript:**
   ```js
   const eventSource = new EventSource('http://localhost:4173/api/run/stream');
   eventSource.onmessage = (event) => {
     const data = JSON.parse(event.data);
     console.log('Event received:', data.type, data);
   };
   ```
