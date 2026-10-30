package worker;

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.util.ArrayList;
import java.util.List;
import java.util.function.BiFunction;

/**
 * Plain-Java worker SDK for the Rheovela Worker HTTP API (rheo serve).
 *
 * <p>Uses only {@link java.net.http.HttpClient} from {@code java.base} — no
 * external dependencies, JDK 11+. A non-2xx response raises
 * {@link WorkerException}.
 */
public class Worker {

    public record Approval(String state, String pending_stage, String final_args,
            String reason, String requested_by, String requested_at, String decided_by,
            String decided_at, String comment, String checkpoint_ref, String effect_cursor) {}

    public record WorkItem(String id, String instance_id, String activity_id, String state,
            String result, String checkpoint_ref, String effect_cursor,
            int attempts, int max_attempts, String last_error, Approval approval) {}

    public record ApprovalRequest(String actor, String reason, String final_args,
            String checkpoint_ref, String effect_cursor) {}

    public record Result(String status, String error, String outcome, String result,
            String checkpoint_ref, String effect_cursor, ApprovalRequest approval) {
        public static Result ok() {
            return new Result("success", null, null, null, null, null, null);
        }

        public static Result success(String result, String checkpoint_ref, String effect_cursor) {
            return new Result("success", null, null, result, checkpoint_ref, effect_cursor, null);
        }

        public static Result fail(String error) {
            return new Result("failure", error, null, null, null, null, null);
        }

        public static Result approval(ApprovalRequest req) {
            return new Result("approval", null, null, null, null, null, req);
        }
    }

    public static class WorkerException extends Exception {
        public final int status;
        public final String body;

        public WorkerException(int status, String body) {
            super("HTTP " + status + ": " + body);
            this.status = status;
            this.body = body;
        }
    }

    private final String baseUrl;
    private final HttpClient http;
    private String claimActor;
    private String claimToolsJson;
    private String claimCredential;

    /** Configure optional claim guards used by {@link #processOnce}. */
    public Worker withClaimOptions(String actor, String toolsJson, String credential) {
        this.claimActor = actor;
        this.claimToolsJson = toolsJson;
        this.claimCredential = credential;
        return this;
    }

    public Worker(String baseUrl) {
        this.baseUrl = baseUrl.endsWith("/")
                ? baseUrl.substring(0, baseUrl.length() - 1)
                : baseUrl;
        this.http = HttpClient.newHttpClient();
    }

    /** GET /api/v1/work-items[?instance=] — returns the ready work items. */
    public List<WorkItem> poll(String instanceId) throws Exception {
        String path = "/api/v1/work-items";
        if (instanceId != null && !instanceId.isEmpty()) {
            path += "?instance=" + instanceId;
        }
        return parseItems(getRaw(path));
    }

    /** GET /api/v1/work-items/{id} — returns a single work item. */
    public WorkItem get(String id) throws Exception {
        return parseItem(getRaw("/api/v1/work-items/" + id));
    }

    /** POST /api/v1/work-items/{id}/claim — returns the lease token. */
    public String claim(String id, String lease) throws Exception {
        return claim(id, lease, null, null, null);
    }

    /** POST claim with optional actor/tools/credential guards. */
    public String claim(String id, String lease, String actor, String toolsJson,
            String credential) throws Exception {
        StringBuilder body = new StringBuilder();
        body.append("{\"lease\":\"").append(jsonEscape(lease)).append('"');
        if (actor != null && !actor.isEmpty()) {
            body.append(",\"actor\":\"").append(jsonEscape(actor)).append('"');
        }
        if (toolsJson != null && !toolsJson.isEmpty()) {
            body.append(",\"tools\":").append(toolsJson);
        }
        if (credential != null && !credential.isEmpty()) {
            body.append(",\"credential\":\"").append(jsonEscape(credential)).append('"');
        }
        body.append('}');
        String resp = post("/api/v1/work-items/" + id + "/claim", body.toString());
        return jsonStr(resp, "token");
    }

    /** POST /api/v1/work-items/{id}/complete. */
    public void complete(String id, String token) throws Exception {
        complete(id, token, null, null, null);
    }

    /** POST /api/v1/work-items/{id}/complete with optional result and cursors. */
    public void complete(String id, String token, String resultJson,
            String checkpointRef, String effectCursor) throws Exception {
        complete(id, token, resultJson, null, 0, checkpointRef, effectCursor);
    }

    /** Complete with optional result/evidence JSON, cost, and cursors. */
    public void complete(String id, String token, String resultJson, String evidenceJson,
            long cost, String checkpointRef, String effectCursor) throws Exception {
        StringBuilder body = new StringBuilder();
        body.append("{\"token\":\"").append(jsonEscape(token)).append('"');
        if (resultJson != null && !resultJson.isEmpty()) {
            body.append(",\"result\":").append(resultJson);
        }
        if (evidenceJson != null && !evidenceJson.isEmpty()) {
            body.append(",\"evidence\":").append(evidenceJson);
        }
        if (cost != 0) {
            body.append(",\"cost\":").append(cost);
        }
        if (checkpointRef != null && !checkpointRef.isEmpty()) {
            body.append(",\"checkpoint_ref\":\"").append(jsonEscape(checkpointRef)).append('"');
        }
        if (effectCursor != null && !effectCursor.isEmpty()) {
            body.append(",\"effect_cursor\":\"").append(jsonEscape(effectCursor)).append('"');
        }
        body.append('}');
        post("/api/v1/work-items/" + id + "/complete", body.toString());
    }

    /** POST /api/v1/work-items/{id}/request-approval. */
    public void requestApproval(String id, String token, ApprovalRequest req) throws Exception {
        String reason = req.reason() == null ? "" : req.reason();
        StringBuilder body = new StringBuilder();
        body.append("{\"token\":\"").append(jsonEscape(token)).append('"');
        body.append(",\"reason\":\"").append(jsonEscape(reason)).append('"');
        if (req.actor() != null && !req.actor().isEmpty()) {
            body.append(",\"actor\":\"").append(jsonEscape(req.actor())).append('"');
        }
        if (req.final_args() != null && !req.final_args().isEmpty()) {
            body.append(",\"final_args\":").append(req.final_args());
        }
        if (req.checkpoint_ref() != null && !req.checkpoint_ref().isEmpty()) {
            body.append(",\"checkpoint_ref\":\"").append(jsonEscape(req.checkpoint_ref())).append('"');
        }
        if (req.effect_cursor() != null && !req.effect_cursor().isEmpty()) {
            body.append(",\"effect_cursor\":\"").append(jsonEscape(req.effect_cursor())).append('"');
        }
        body.append('}');
        post("/api/v1/work-items/" + id + "/request-approval", body.toString());
    }

    /** POST /api/v1/work-items/{id}/approve. */
    public void approve(String id, String actor, String comment, String finalArgsJson) throws Exception {
        StringBuilder body = new StringBuilder();
        body.append("{\"actor\":\"").append(jsonEscape(actor)).append('"');
        body.append(",\"comment\":\"").append(jsonEscape(comment == null ? "" : comment)).append('"');
        if (finalArgsJson != null && !finalArgsJson.isEmpty()) {
            body.append(",\"final_args\":").append(finalArgsJson);
        }
        body.append('}');
        post("/api/v1/work-items/" + id + "/approve", body.toString());
    }

    /** POST /api/v1/work-items/{id}/reject. */
    public void reject(String id, String actor, String reason) throws Exception {
        post("/api/v1/work-items/" + id + "/reject",
                "{\"actor\":\"" + jsonEscape(actor) + "\",\"reason\":\"" + jsonEscape(reason) + "\"}");
    }

    /** POST /api/v1/work-items/{id}/cancel. */
    public void cancel(String id) throws Exception {
        post("/api/v1/work-items/" + id + "/cancel", "{}");
    }

    /** POST /api/v1/work-items/{id}/fail. */
    public void fail(String id, String token, String error) throws Exception {
        post("/api/v1/work-items/" + id + "/fail",
                "{\"token\":\"" + jsonEscape(token) + "\",\"error\":\"" + jsonEscape(error) + "\"}");
    }

    /**
     * Poll once, claim each item, run {@code fn}, then complete/fail/request
     * approval. Items that cannot be claimed (claimed by another worker) are
     * skipped. Returns the number of items processed.
     */
    public int processOnce(BiFunction<WorkItem, String, Result> fn, String lease) throws Exception {
        int processed = 0;
        for (WorkItem item : poll(null)) {
            String token;
            try {
                token = claim(item.id(), lease, claimActor, claimToolsJson, claimCredential);
            } catch (WorkerException e) {
                continue; // claimed by another worker
            }
            Result res = fn.apply(item, token);
            try {
                if ("success".equals(res.status())) {
                    complete(item.id(), token, res.result(),
                            res.checkpoint_ref(), res.effect_cursor());
                } else if ("approval".equals(res.status())) {
                    ApprovalRequest req = res.approval();
                    if (req == null) {
                        fail(item.id(), token, "approval result is missing approval request");
                    } else {
                        String checkpointRef = req.checkpoint_ref() == null
                                || req.checkpoint_ref().isEmpty()
                                        ? res.checkpoint_ref()
                                        : req.checkpoint_ref();
                        String effectCursor = req.effect_cursor() == null
                                || req.effect_cursor().isEmpty()
                                        ? res.effect_cursor()
                                        : req.effect_cursor();
                        ApprovalRequest merged = new ApprovalRequest(
                                req.actor(), req.reason(), req.final_args(),
                                checkpointRef, effectCursor);
                        try {
                            requestApproval(item.id(), token, merged);
                        } catch (WorkerException e) {
                            fail(item.id(), token, e.getMessage());
                        }
                    }
                } else {
                    fail(item.id(), token, res.error() == null ? "failure" : res.error());
                }
                processed++;
            } catch (WorkerException e) {
                // stale token / expired lease
            }
        }
        return processed;
    }

    // --- HTTP helpers ---

    private String getRaw(String path) throws Exception {
        HttpRequest req = HttpRequest.newBuilder(URI.create(baseUrl + path)).GET().build();
        return send(req);
    }

    private String post(String path, String json) throws Exception {
        HttpRequest req = HttpRequest.newBuilder(URI.create(baseUrl + path))
                .header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofString(json))
                .build();
        return send(req);
    }

    private String send(HttpRequest req) throws Exception {
        HttpResponse<String> resp = http.send(req, HttpResponse.BodyHandlers.ofString());
        if (resp.statusCode() < 200 || resp.statusCode() >= 300) {
            throw new WorkerException(resp.statusCode(), resp.body());
        }
        return resp.body();
    }

    // --- minimal JSON helpers (no external deps) ---

    private static List<WorkItem> parseItems(String body) {
        List<WorkItem> items = new ArrayList<>();
        if (body == null) {
            return items;
        }
        int i = 0;
        while (i < body.length()) {
            if (body.charAt(i) == '{') {
                int end = matchingBracket(body, i);
                if (end < 0) {
                    break;
                }
                String obj = body.substring(i, end + 1);
                if (jsonStr(obj, "id") != null) {
                    items.add(parseItem(obj));
                }
                i = end + 1;
            } else {
                i++;
            }
        }
        return items;
    }

    private static WorkItem parseItem(String obj) {
        return new WorkItem(
                jsonStr(obj, "id"),
                jsonStr(obj, "instance_id"),
                jsonStr(obj, "activity_id"),
                jsonStr(obj, "state"),
                jsonValue(obj, "result"),
                jsonStr(obj, "checkpoint_ref"),
                jsonStr(obj, "effect_cursor"),
                jsonInt(obj, "attempts"),
                jsonInt(obj, "max_attempts"),
                jsonStr(obj, "last_error"),
                parseApproval(jsonObject(obj, "approval")));
    }

    private static Approval parseApproval(String obj) {
        if (obj == null) {
            return null;
        }
        return new Approval(
                jsonStr(obj, "state"),
                jsonStr(obj, "pending_stage"),
                jsonValue(obj, "final_args"),
                jsonStr(obj, "reason"),
                jsonStr(obj, "requested_by"),
                jsonStr(obj, "requested_at"),
                jsonStr(obj, "decided_by"),
                jsonStr(obj, "decided_at"),
                jsonStr(obj, "comment"),
                jsonStr(obj, "checkpoint_ref"),
                jsonStr(obj, "effect_cursor"));
    }

    /** Returns the raw JSON value (quoted string, object, array, or literal). */
    private static String jsonValue(String body, String key) {
        if (body == null) {
            return null;
        }
        int colon = indexOfKey(body, key);
        if (colon < 0) {
            return null;
        }
        int start = colon + 1;
        while (start < body.length() && Character.isWhitespace(body.charAt(start))) {
            start++;
        }
        if (start >= body.length()) {
            return null;
        }
        char c = body.charAt(start);
        if (c == '"') {
            int end = start + 1;
            while (end < body.length()) {
                char ch = body.charAt(end);
                if (ch == '\\') {
                    end += 2;
                    continue;
                }
                if (ch == '"') {
                    end++;
                    break;
                }
                end++;
            }
            return body.substring(start, end);
        }
        if (c == '{' || c == '[') {
            int end = matchingBracket(body, start);
            return end < 0 ? null : body.substring(start, end + 1);
        }
        int end = start;
        while (end < body.length()) {
            char ch = body.charAt(end);
            if (ch == ',' || ch == '}' || ch == ']' || Character.isWhitespace(ch)) {
                break;
            }
            end++;
        }
        return body.substring(start, end);
    }

    /** Returns the raw nested JSON object for {@code key}, or null. */
    private static String jsonObject(String body, String key) {
        String raw = jsonValue(body, key);
        if (raw != null && raw.startsWith("{")) {
            return raw;
        }
        return null;
    }

    private static String jsonStr(String body, String key) {
        String raw = jsonValue(body, key);
        if (raw == null) {
            return null;
        }
        if (raw.length() >= 2 && raw.charAt(0) == '"' && raw.charAt(raw.length() - 1) == '"') {
            return unescape(raw.substring(1, raw.length() - 1));
        }
        return raw;
    }

    private static int jsonInt(String body, String key) {
        String raw = jsonValue(body, key);
        if (raw == null || raw.isEmpty()) {
            return 0;
        }
        try {
            return Integer.parseInt(raw.trim());
        } catch (NumberFormatException e) {
            return 0;
        }
    }

    /** Finds {@code "key":} at the top level of a JSON object. Returns the colon index. */
    private static int indexOfKey(String body, String key) {
        int depth = 0;
        for (int i = 0; i < body.length(); i++) {
            char c = body.charAt(i);
            if (c == '"') {
                int end = i + 1;
                StringBuilder sb = new StringBuilder();
                while (end < body.length()) {
                    char ch = body.charAt(end);
                    if (ch == '\\') {
                        if (end + 1 < body.length()) {
                            sb.append(body.charAt(end + 1));
                        }
                        end += 2;
                        continue;
                    }
                    if (ch == '"') {
                        break;
                    }
                    sb.append(ch);
                    end++;
                }
                int colon = end + 1;
                while (colon < body.length() && Character.isWhitespace(body.charAt(colon))) {
                    colon++;
                }
                if (depth == 1 && colon < body.length() && body.charAt(colon) == ':'
                        && sb.toString().equals(key)) {
                    return colon;
                }
                i = end;
            } else if (c == '{' || c == '[') {
                depth++;
            } else if (c == '}' || c == ']') {
                depth--;
            }
        }
        return -1;
    }

    private static int matchingBracket(String s, int start) {
        char open = s.charAt(start);
        char close = open == '{' ? '}' : ']';
        int depth = 0;
        boolean inStr = false;
        for (int i = start; i < s.length(); i++) {
            char c = s.charAt(i);
            if (inStr) {
                if (c == '\\') {
                    i++;
                } else if (c == '"') {
                    inStr = false;
                }
                continue;
            }
            if (c == '"') {
                inStr = true;
            } else if (c == open) {
                depth++;
            } else if (c == close) {
                depth--;
                if (depth == 0) {
                    return i;
                }
            }
        }
        return -1;
    }

    private static String unescape(String s) {
        StringBuilder sb = new StringBuilder(s.length());
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            if (c == '\\' && i + 1 < s.length()) {
                char n = s.charAt(++i);
                switch (n) {
                    case 'n' -> sb.append('\n');
                    case 'r' -> sb.append('\r');
                    case 't' -> sb.append('\t');
                    case 'b' -> sb.append('\b');
                    case 'f' -> sb.append('\f');
                    case 'u' -> {
                        if (i + 4 < s.length()) {
                            sb.append((char) Integer.parseInt(s.substring(i + 1, i + 5), 16));
                            i += 4;
                        }
                    }
                    default -> sb.append(n);
                }
            } else {
                sb.append(c);
            }
        }
        return sb.toString();
    }

    private static String jsonEscape(String s) {
        if (s == null) {
            return "";
        }
        StringBuilder sb = new StringBuilder(s.length() + 8);
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            switch (c) {
                case '"' -> sb.append("\\\"");
                case '\\' -> sb.append("\\\\");
                case '\b' -> sb.append("\\b");
                case '\f' -> sb.append("\\f");
                case '\n' -> sb.append("\\n");
                case '\r' -> sb.append("\\r");
                case '\t' -> sb.append("\\t");
                default -> {
                    if (c < 0x20) {
                        sb.append(String.format("\\u%04x", (int) c));
                    } else {
                        sb.append(c);
                    }
                }
            }
        }
        return sb.toString();
    }
}
