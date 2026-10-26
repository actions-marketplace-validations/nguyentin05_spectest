import { http, HttpResponse } from "msw";
import { healthFixture } from "../fixtures";

export const healthHandlers = [http.get("/api/health", () => HttpResponse.json(healthFixture))];
