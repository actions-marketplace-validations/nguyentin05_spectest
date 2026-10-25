import type {
  HealthResponse,
  HistoryResponse,
  LLMConfig,
  PlanResponse,
  WorkspaceItem,
} from "../api/types";

export const workspaceDir = "/home/tin/Project/current/spectest";

export const healthFixture: HealthResponse = {
  status: "ok",
  time: "2026-10-25T13:45:00+07:00",
  work_dir: workspaceDir,
};

export function createConfigFixture(): LLMConfig {
  return {
    provider: "gemini",
    model: "gemini-1.5-flash",
    endpoint: "https://generativelanguage.googleapis.com",
    has_api_key: true,
    api_key_masked: "AIzaSy...4xK9",
  };
}

export function createWorkspaceFixture(): WorkspaceItem[] {
  return [
    {
      name: "specs/auth-api.yaml",
      path: `${workspaceDir}/specs/auth-api.yaml`,
      type: "openapi",
      size: 18450,
    },
    {
      name: "specs/order-api.json",
      path: `${workspaceDir}/specs/order-api.json`,
      type: "openapi",
      size: 24102,
    },
    {
      name: "docs/requirements.md",
      path: `${workspaceDir}/docs/requirements.md`,
      type: "prd",
      size: 6512,
    },
    {
      name: "tests/spectest.yaml",
      path: `${workspaceDir}/tests/spectest.yaml`,
      type: "suite",
      size: 8940,
    },
    {
      name: "tests/spectest.log.yaml",
      path: `${workspaceDir}/tests/spectest.log.yaml`,
      type: "history",
      size: 14210,
    },
  ];
}

export function createPlanFixture(): PlanResponse {
  return {
    path: "tests/spectest.yaml",
    version: 1,
    servers: ["auth", "order"],
    apis: [
      "POST {auth}/api/register",
      "POST {auth}/api/login",
      "POST {order}/api/products",
      "POST {order}/api/orders",
    ],
    analysis:
      "- Có register + login nên tạo được user mới độc lập cho từng scenario.\n- Requirements có sẵn tài khoản admin dùng tạo product.",
    seed: {
      admin: {
        vars: {
          admin_email: "admin@shop.com",
          admin_password: "Admin@123",
        },
      },
    },
    setup: {
      admin_session: {
        vars: {
          email: "{{admin_email}}",
          password: "{{admin_password}}",
        },
        steps: [
          {
            call: "POST {auth}/api/login",
            headers: {},
            body: {
              email: "{{email}}",
              password: "{{password}}",
            },
            expect: 200,
            save: { admin_token: "body.access_token" },
          },
        ],
      },
      user: {
        vars: {
          email: "user_{{random}}@test.com",
          password: "P@ssw0rd123",
        },
        steps: [
          {
            call: "POST {auth}/api/register",
            body: {
              email: "{{email}}",
              password: "{{password}}",
            },
            expect: 201,
            save: { user_id: "body.id" },
          },
        ],
      },
      logged_in_user: {
        use: ["user"],
        steps: [
          {
            call: "POST {auth}/api/login",
            body: {
              email: "{{email}}",
              password: "{{password}}",
            },
            expect: 200,
            save: { user_token: "body.access_token" },
          },
        ],
      },
    },
    scenarios: [
      {
        id: "login-wrong-password",
        name: "Login sai password",
        setup: ["user"],
        steps: [
          {
            call: "POST {auth}/api/login",
            body: {
              email: "{{email}}",
              password: "sai_mat_khau",
            },
            expect: 401,
          },
        ],
        locked: false,
        skip: "",
      },
      {
        id: "order-over-stock",
        name: "Đặt hàng vượt tồn kho",
        setup: ["logged_in_user"],
        steps: [
          {
            call: "POST {order}/api/orders",
            headers: {
              Authorization: "Bearer {{user_token}}",
            },
            body: {
              items: [{ product_id: "prod_1", quantity: 999 }],
            },
            expect: 422,
          },
        ],
        locked: true,
        skip: "",
      },
    ],
    stats: {
      total_scenarios: 2,
      total_setups: 3,
      total_steps: 4,
      locked_scenarios: 1,
      skipped_scenarios: 0,
    },
  };
}

export function createHistoryFixture(): HistoryResponse {
  return {
    base: {
      at: "2026-10-20T09:00:00+07:00",
      by: "ai",
      after_hash: "5c1d02a9b7e4",
      spectest_summary: {
        servers: ["auth", "order"],
        apis_count: 4,
        scenarios_count: 2,
      },
    },
    updates: [
      {
        n: 1,
        at: "2026-10-24T10:15:00+07:00",
        by: "ai",
        note: "API added: POST {order}/api/orders/{id}/cancel",
        trigger: {
          openapi: {
            added: ["POST {order}/api/orders/{id}/cancel"],
            changed: ["POST {auth}/api/register"],
            removed: [],
          },
          requirements: {
            before_hash: "a1b2c3d4e5f6",
            after_hash: "f6e5d4c3b2a1",
          },
        },
        create: {
          setup_added: ["order"],
          scenarios_added: ["order-cancel-pending"],
        },
        change: [
          {
            target: "setup",
            id: "user",
            reason: "register bổ sung field phone bắt buộc",
            before: { vars: { email: "user_{{random}}@test.com" } },
            after: {
              vars: { email: "user_{{random}}@test.com", phone: "0912345678" },
            },
          },
        ],
        after_hash: "8e41ab77c9d0",
      },
    ],
  };
}

export function createMarkdownFixtures(): Record<string, string> {
  return {
    "docs/requirements.md": [
      "# Product Requirements Document",
      "",
      "## 1. Authentication",
      "User must register with a unique email and a password of at least 8 characters.",
      "",
      "## 2. Orders",
      "An order must be rejected with 422 when the requested quantity exceeds stock.",
      "",
      "## 3. Accounts",
      "An admin account is seeded to create products before the suite runs.",
      "",
    ].join("\n"),
  };
}

export function createLogsFixture(): string[] {
  return [
    "[13:40:01] SpecTest Web UI Server listening on :4173",
    "[13:41:05] Uploaded file: auth-service.yaml",
    "[13:42:10] Test run scheduled against http://localhost:8080",
  ];
}
