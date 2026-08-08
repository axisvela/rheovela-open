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

    public record WorkItem(String id, String instance_id, String activity_id, String state) {}

    public record Result(String status, String error) {
        public static Result ok() { return new Result("success", null); }
        public static Result fail(String error) { return new Result("failure", error); }
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
        return parseItems(get(path));
    }

    /** POST /api/v1/work-items/{id}/claim — returns the lease token. */
    public String claim(String id, String lease) throws Exception {
        String body = post("/api/v1/work-items/" + id + "/claim",
                "{\"lease\":\"" + lease + "\"}");
        return jsonStr(body, "token");
    }

    /** POST /api/v1/work-items/{id}/complete. */
    public void complete(String id, String token) throws Exception {
        post("/api/v1/work-items/" + id + "/complete",
                "{\"token\":\"" + token + "\"}");
    }

    /** POST /api/v1/work-items/{id}/fail. */
    public void fail(String id, String token, String error) throws Exception {
        post("/api/v1/work-items/" + id + "/fail",
                "{\"token\":\"" + token + "\",\"error\":\"" + error + "\"}");
    }

    /**
     * Poll once, claim each item, run {@code fn}, then complete or fail.
     * Items that cannot be claimed (claimed by another worker) are skipped.
     * Returns the number of items processed to completion.
     */
    public int processOnce(BiFunction<WorkItem, String, Result> fn, String lease) throws Exception {
        int processed = 0;
        for (WorkItem item : poll(null)) {
            String token;
            try {
                token = claim(item.id(), lease);
            } catch (WorkerException e) {
                continue; // claimed by another worker
            }
            Result res = fn.apply(item, token);
            try {
                if ("success".equals(res.status())) {
                    complete(item.id(), token);
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

    private String get(String path) throws Exception {
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
        int i = 0;
        while (i < body.length()) {
            if (body.charAt(i) == '{') {
                int depth = 0;
                int j = i;
                for (; j < body.length(); j++) {
                    char c = body.charAt(j);
                    if (c == '{') {
                        depth++;
                    } else if (c == '}') {
                        depth--;
                        if (depth == 0) {
                            j++;
                            break;
                        }
                    }
                }
                String obj = body.substring(i, j);
                String id = jsonStr(obj, "id");
                if (id != null) {
                    items.add(new WorkItem(id,
                            jsonStr(obj, "instance_id"),
                            jsonStr(obj, "activity_id"),
                            jsonStr(obj, "state")));
                }
                i = j;
            } else {
                i++;
            }
        }
        return items;
    }

    private static String jsonStr(String body, String key) {
        String target = "\"" + key + "\"";
        int idx = body.indexOf(target);
        if (idx < 0) {
            return null;
        }
        idx = body.indexOf(':', idx + target.length());
        if (idx < 0) {
            return null;
        }
        int start = idx + 1;
        while (start < body.length() && Character.isWhitespace(body.charAt(start))) {
            start++;
        }
        if (start < body.length() && body.charAt(start) == '"') {
            int end = start + 1;
            while (end < body.length() && body.charAt(end) != '"') {
                end++;
            }
            return body.substring(start + 1, end);
        }
        return null;
    }
}
