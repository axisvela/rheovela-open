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
 * Runnable demo. Serves a fake Worker HTTP API in-process, runs one poll cycle
 * through {@link Worker#processOnce}, and prints the outcome.
 */
public class Example {

    public static void main(String[] args) throws Exception {
        FakeApi api = new FakeApi();
        api.start();

        Worker w = new Worker("http://127.0.0.1:" + api.port());
        int processed = w.processOnce((item, token) -> {
            if ("task-1".equals(item.id())) {
                return Worker.Result.ok();
            }
            return Worker.Result.fail("boom");
        }, "30s");

        System.out.println("processed=" + processed);
        for (String rec : api.records()) {
            System.out.println(rec);
        }

        api.stop();
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

            if ("GET".equals(method) && path.equals("/api/v1/work-items")) {
                String body = "[{\"id\":\"task-1\",\"instance_id\":\"inst-1\",\"activity_id\":\"act-1\",\"state\":\"ready\"},"
                        + "{\"id\":\"task-2\",\"instance_id\":\"inst-1\",\"activity_id\":\"act-1\",\"state\":\"ready\"}]";
                respond(ex, 200, body);
            } else if ("POST".equals(method) && path.endsWith("/claim")) {
                respond(ex, 200, "{\"token\":\"tok-1\",\"lease\":\"30s\"}");
            } else if ("POST".equals(method) && path.endsWith("/complete")) {
                records.add(idFrom(path, "/complete") + ": done");
                respond(ex, 200, "{\"ok\":true}");
            } else if ("POST".equals(method) && path.endsWith("/fail")) {
                records.add(idFrom(path, "/fail") + ": failed");
                respond(ex, 200, "{\"ok\":true}");
            } else {
                respond(ex, 404, "{\"error\":\"not found\"}");
            }
        }

        private static String idFrom(String path, String suffix) {
            String p = path.substring(0, path.length() - suffix.length());
            return p.substring(p.lastIndexOf('/') + 1);
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
