package example;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;

import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;

import worker.Worker;

/**
 * Runnable demo + contract check. Serves a fake Worker HTTP API in-process,
 * exercises the v1.4.0 client surface, and fails if the recorded requests do
 * not match the expected endpoint/body sequence.
 */
public class Example {

    private static final String TASK1 = "{"
            + "\"id\":\"task-1\",\"instance_id\":\"inst-1\",\"activity_id\":\"act-1\","
            + "\"state\":\"ready\",\"result\":{\"previous\":true},"
            + "\"checkpoint_ref\":\"cp-0\",\"effect_cursor\":\"cur-0\","
            + "\"attempts\":2,\"max_attempts\":5,\"last_error\":\"retry\"}";

    private static final String ITEMS = "["
            + TASK1 + ","
            + "{\"id\":\"task-2\",\"instance_id\":\"inst-1\",\"activity_id\":\"act-1\","
            + "\"state\":\"awaiting_approval\",\"attempts\":0,\"max_attempts\":1,"
            + "\"last_error\":\"\",\"approval\":{"
            + "\"state\":\"pending\",\"pending_stage\":\"review\","
            + "\"final_args\":{\"seed\":1},\"reason\":\"needs review\","
            + "\"requested_by\":\"worker\",\"requested_at\":\"t0\","
            + "\"decided_by\":\"\",\"decided_at\":\"\",\"comment\":\"\","
            + "\"checkpoint_ref\":\"cp-req\",\"effect_cursor\":\"cur-req\"}}"
            + "]";

    public static void main(String[] args) throws Exception {
        FakeApi api = new FakeApi();
        api.start();
        try {
            Worker w = new Worker("http://127.0.0.1:" + api.port());

            int processed = w.processOnce((item, token) -> {
                if ("task-1".equals(item.id())) {
                    check("{\"previous\":true}".equals(item.result()), "task-1 result");
                    check("cp-0".equals(item.checkpoint_ref()), "task-1 checkpoint_ref");
                    check("cur-0".equals(item.effect_cursor()), "task-1 effect_cursor");
                    check(item.attempts() == 2, "task-1 attempts");
                    check(item.max_attempts() == 5, "task-1 max_attempts");
                    check("retry".equals(item.last_error()), "task-1 last_error");
                    check(item.approval() == null, "task-1 approval");
                    return Worker.Result.success("{\"answer\":42}", "cp-1", "cur-1");
                }
                if ("task-2".equals(item.id())) {
                    Worker.Approval approval = item.approval();
                    check(approval != null, "task-2 approval present");
                    check("awaiting_approval".equals(item.state()), "task-2 state");
                    check("pending".equals(approval.state()), "task-2 approval.state");
                    check("review".equals(approval.pending_stage()), "task-2 approval.pending_stage");
                    check("{\"seed\":1}".equals(approval.final_args()), "task-2 approval.final_args");
                    check("worker".equals(approval.requested_by()), "task-2 approval.requested_by");
                    Worker.ApprovalRequest req = new Worker.ApprovalRequest(
                            "worker", "needs review", "{\"plan\":\"b\"}", "", "");
                    return new Worker.Result("approval", null, null, null,
                            "cp-res", "cur-res", req);
                }
                return Worker.Result.fail("unexpected item " + item.id());
            }, "30s");

            check(processed == 2, "processed expected 2 but was " + processed);

            Worker.WorkItem fetched = w.get("task-1");
            check(fetched != null && "task-1".equals(fetched.id()), "get id");
            check(fetched.attempts() == 2, "get attempts");

            w.approve("task-2", "alice", "lgtm", "{\"plan\":\"c\"}");
            w.reject("task-3", "bob", "nope");
            w.cancel("task-4");

            List<String> expected = List.of(
                    "GET /api/v1/work-items",
                    "POST /api/v1/work-items/task-1/claim {\"lease\":\"30s\"}",
                    "POST /api/v1/work-items/task-1/complete {\"token\":\"tok-1\",\"result\":{\"answer\":42},\"checkpoint_ref\":\"cp-1\",\"effect_cursor\":\"cur-1\"}",
                    "POST /api/v1/work-items/task-2/claim {\"lease\":\"30s\"}",
                    "POST /api/v1/work-items/task-2/request-approval {\"token\":\"tok-1\",\"reason\":\"needs review\",\"actor\":\"worker\",\"final_args\":{\"plan\":\"b\"},\"checkpoint_ref\":\"cp-res\",\"effect_cursor\":\"cur-res\"}",
                    "GET /api/v1/work-items/task-1",
                    "POST /api/v1/work-items/task-2/approve {\"actor\":\"alice\",\"comment\":\"lgtm\",\"final_args\":{\"plan\":\"c\"}}",
                    "POST /api/v1/work-items/task-3/reject {\"actor\":\"bob\",\"reason\":\"nope\"}",
                    "POST /api/v1/work-items/task-4/cancel {}");

            check(expected.equals(api.records()),
                    "records mismatch\nexpected=" + expected + "\nactual=" + api.records());
        } finally {
            api.stop();
        }
        System.out.println("ok");
    }

    private static void check(boolean ok, String message) {
        if (!ok) {
            throw new AssertionError(message);
        }
    }

    /** In-process fake of the core Worker HTTP API. */
    static class FakeApi {
        private HttpServer server;
        private final List<String> records = new ArrayList<>();

        void start() throws IOException {
            server = HttpServer.create(new InetSocketAddress(0), 0);
            server.createContext("/", this::handle);
            server.start();
        }

        int port() {
            return server.getAddress().getPort();
        }

        void stop() {
            server.stop(0);
        }

        List<String> records() {
            return records;
        }

        private void handle(HttpExchange ex) throws IOException {
            String path = ex.getRequestURI().getPath();
            String method = ex.getRequestMethod();
            String body = new String(ex.getRequestBody().readAllBytes(), StandardCharsets.UTF_8);

            String record = method + " " + path;
            if (!body.isEmpty()) {
                record += " " + body;
            }
            records.add(record);

            if ("GET".equals(method) && path.equals("/api/v1/work-items")) {
                respond(ex, 200, ITEMS);
            } else if ("GET".equals(method) && path.startsWith("/api/v1/work-items/")) {
                respond(ex, 200, TASK1);
            } else if ("POST".equals(method) && path.endsWith("/claim")) {
                respond(ex, 200, "{\"token\":\"tok-1\",\"lease\":\"30s\"}");
            } else if ("POST".equals(method) && (path.endsWith("/complete")
                    || path.endsWith("/request-approval")
                    || path.endsWith("/approve")
                    || path.endsWith("/reject")
                    || path.endsWith("/cancel")
                    || path.endsWith("/fail"))) {
                respond(ex, 200, "{\"ok\":true}");
            } else {
                respond(ex, 404, "{\"error\":\"not found\"}");
            }
        }

        private static void respond(HttpExchange ex, int code, String body) throws IOException {
            byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
            ex.sendResponseHeaders(code, bytes.length);
            try (OutputStream os = ex.getResponseBody()) {
                os.write(bytes);
            }
        }
    }
}
