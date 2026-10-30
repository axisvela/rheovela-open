package worker;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.WebSocket;
import java.nio.channels.FileChannel;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.nio.file.StandardOpenOption;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.CompletionStage;
import java.util.concurrent.TimeUnit;

/**
 * Durable WebSocket event client for Rheovela (JDK 11+ only).
 *
 * <p>Mirrors the Go/Python/TypeScript clients: a file-backed cursor and pending
 * queue, {@code after_seq} reconnect, exponential backoff, and at-least-once
 * handler delivery.
 */
public final class Events {

    private Events() {}

    public record Event(long seq, String type, String actorId, String payload,
            String wallTime, String raw) {}

    @FunctionalInterface
    public interface Handler {
        void handle(Event event) throws Exception;
    }

    public static final class DurableQueue {
        private final Path sequencePath;
        private final Path pendingPath;
        private long lastSeq;
        private final List<Event> pending = new ArrayList<>();

        public DurableQueue(Path directory) throws IOException {
            Files.createDirectories(directory);
            this.sequencePath = directory.resolve("events-last-seq");
            this.pendingPath = directory.resolve("events-pending.jsonl");
            load();
        }

        public synchronized long lastSeq() {
            return lastSeq;
        }

        public synchronized List<Event> pending() {
            return new ArrayList<>(pending);
        }

        public synchronized void push(Event event) throws IOException {
            if (event.seq() <= lastSeq || pending.stream().anyMatch(e -> e.seq() == event.seq())) {
                return;
            }
            pending.add(event);
            pending.sort(Comparator.comparingLong(Event::seq));
            Files.writeString(pendingPath, event.raw() + "\n", StandardCharsets.UTF_8,
                    StandardOpenOption.CREATE, StandardOpenOption.APPEND);
            try (FileChannel channel = FileChannel.open(pendingPath, StandardOpenOption.WRITE)) {
                channel.force(true);
            }
        }

        public synchronized void ack(long seq) throws IOException {
            if (seq > lastSeq) {
                lastSeq = seq;
            }
            pending.removeIf(event -> event.seq() <= seq);
            rewritePending();
            writeSequence();
        }

        public synchronized void setFloor(long seq) throws IOException {
            if (!pending.isEmpty() || seq <= lastSeq) {
                return;
            }
            lastSeq = seq;
            writeSequence();
        }

        private void load() throws IOException {
            if (Files.exists(sequencePath)) {
                String raw = Files.readString(sequencePath, StandardCharsets.UTF_8).trim();
                if (!raw.isEmpty()) {
                    lastSeq = Long.parseLong(raw);
                }
            }
            if (!Files.exists(pendingPath)) {
                return;
            }
            try (BufferedReader reader = Files.newBufferedReader(pendingPath, StandardCharsets.UTF_8)) {
                String line;
                while ((line = reader.readLine()) != null) {
                    if (line.isBlank()) {
                        continue;
                    }
                    Event event = parse(line);
                    if (event.seq() > lastSeq) {
                        pending.add(event);
                    }
                }
            }
            pending.sort(Comparator.comparingLong(Event::seq));
        }

        private void rewritePending() throws IOException {
            StringBuilder out = new StringBuilder();
            for (Event event : pending) {
                out.append(event.raw()).append('\n');
            }
            Path tmp = pendingPath.resolveSibling(pendingPath.getFileName() + ".tmp");
            Files.writeString(tmp, out.toString(), StandardCharsets.UTF_8);
            Files.move(tmp, pendingPath, StandardCopyOption.REPLACE_EXISTING, StandardCopyOption.ATOMIC_MOVE);
        }

        private void writeSequence() throws IOException {
            Path tmp = sequencePath.resolveSibling(sequencePath.getFileName() + ".tmp");
            Files.writeString(tmp, Long.toString(lastSeq), StandardCharsets.UTF_8);
            Files.move(tmp, sequencePath, StandardCopyOption.REPLACE_EXISTING, StandardCopyOption.ATOMIC_MOVE);
        }
    }

    public static final class Client implements AutoCloseable {
        private final String url;
        private final String streamId;
        private final List<String> types;
        private final String correlationKey;
        private final String predicate;
        private final Handler handler;
        private final java.util.function.Consumer<Exception> onError;
        private final long reconnectMinMs;
        private final long reconnectMaxMs;
        private final long retryDelayMs;
        private final DurableQueue queue;
        private volatile boolean running = true;
        private volatile WebSocket socket;
        private final HttpClient http = HttpClient.newHttpClient();

        public Client(String url, String streamId, Path stateDir, Handler handler,
                List<String> types, String correlationKey, String predicate, long afterSeq,
                long reconnectMinMs, long reconnectMaxMs, long retryDelayMs,
                java.util.function.Consumer<Exception> onError) throws IOException {
            if (url == null || url.isEmpty()) throw new IllegalArgumentException("url is required");
            if (streamId == null || streamId.isEmpty()) throw new IllegalArgumentException("streamId is required");
            this.url = url;
            this.streamId = streamId;
            this.types = types == null ? List.of() : List.copyOf(types);
            this.correlationKey = correlationKey == null ? "" : correlationKey;
            this.predicate = predicate == null ? "" : predicate;
            this.handler = handler;
            this.onError = onError;
            this.reconnectMinMs = reconnectMinMs > 0 ? reconnectMinMs : 200;
            this.reconnectMaxMs = reconnectMaxMs > 0 ? reconnectMaxMs : 30_000;
            this.retryDelayMs = retryDelayMs > 0 ? retryDelayMs : 1_000;
            this.queue = new DurableQueue(stateDir);
            if (afterSeq > 0) {
                queue.setFloor(afterSeq);
            }
        }

        public long lastSeq() {
            return queue.lastSeq();
        }

        public List<Event> pending() {
            return queue.pending();
        }

        public void run() {
            long backoff = reconnectMinMs;
            while (running) {
                try {
                    drain();
                    session();
                    backoff = reconnectMinMs;
                } catch (Exception error) {
                    if (!running) return;
                    if (onError != null) onError.accept(error);
                    sleep(backoff);
                    backoff = Math.min(backoff * 2, reconnectMaxMs);
                }
            }
        }

        private void drain() throws Exception {
            for (Event event : queue.pending()) {
                handler.handle(event);
                queue.ack(event.seq());
            }
        }

        private void session() throws Exception {
            CompletableFuture<Void> done = new CompletableFuture<>();
            StringBuilder partial = new StringBuilder();
            WebSocket.Listener listener = new WebSocket.Listener() {
                @Override
                public void onOpen(WebSocket webSocket) {
                    webSocket.request(1);
                }

                @Override
                public CompletionStage<?> onText(WebSocket webSocket, CharSequence data, boolean last) {
                    partial.append(data);
                    if (last) {
                        String raw = partial.toString();
                        partial.setLength(0);
                        try {
                            Event event = parse(raw);
                            if (event.seq() > 0) {
                                queue.push(event);
                                handler.handle(event);
                                queue.ack(event.seq());
                                webSocket.sendText("{\"type\":\"ack\",\"seq\":" + event.seq() + "}", true);
                            }
                        } catch (Exception error) {
                            webSocket.abort();
                            done.completeExceptionally(error);
                            return null;
                        }
                    }
                    webSocket.request(1);
                    return null;
                }

                @Override
                public CompletionStage<?> onClose(WebSocket webSocket, int statusCode, String reason) {
                    done.complete(null);
                    return null;
                }

                @Override
                public void onError(WebSocket webSocket, Throwable error) {
                    done.completeExceptionally(error);
                }
            };
            WebSocket webSocket = http.newWebSocketBuilder()
                    .buildAsync(URI.create(endpoint()), listener)
                    .get(10, TimeUnit.SECONDS);
            this.socket = webSocket;
            done.get();
            this.socket = null;
        }

        private String endpoint() {
            URI parsed = URI.create(url);
            String scheme = parsed.getScheme() == null ? "ws" : parsed.getScheme();
            if (scheme.equals("https")) scheme = "wss";
            if (scheme.equals("http")) scheme = "ws";
            String path = parsed.getRawPath() == null ? "" : parsed.getRawPath();
            path = path.replaceAll("/$", "") + "/api/v1/events/ws";
            StringBuilder query = new StringBuilder("stream=").append(encode(streamId));
            if (queue.lastSeq() > 0) query.append("&after_seq=").append(queue.lastSeq());
            if (!types.isEmpty()) query.append("&types=").append(encode(String.join(",", types)));
            if (!correlationKey.isEmpty()) query.append("&correlation_key=").append(encode(correlationKey));
            if (!predicate.isEmpty()) query.append("&predicate=").append(encode(predicate));
            if (parsed.getRawQuery() != null && !parsed.getRawQuery().isEmpty()) {
                query.append('&').append(parsed.getRawQuery());
            }
            return scheme + "://" + parsed.getRawAuthority() + path + "?" + query;
        }

        @Override
        public void close() {
            running = false;
            WebSocket current = socket;
            if (current != null) {
                try {
                    current.abort();
                } catch (Exception ignored) {
                    // best effort
                }
            }
        }

        private static void sleep(long ms) {
            try {
                Thread.sleep(ms);
            } catch (InterruptedException interrupted) {
                Thread.currentThread().interrupt();
            }
        }
    }

    public static Event parse(String raw) {
        long seq = parseLong(raw, "seq");
        return new Event(seq, parseString(raw, "type"), parseString(raw, "actor_id"),
                rawValue(raw, "payload"), parseString(raw, "wall_time"), raw);
    }

    private static long parseLong(String json, String key) {
        String value = rawValue(json, key);
        if (value == null || value.isBlank()) return 0;
        try {
            return Long.parseLong(value.trim());
        } catch (NumberFormatException error) {
            return 0;
        }
    }

    private static String parseString(String json, String key) {
        String value = rawValue(json, key);
        if (value == null || value.length() < 2 || value.charAt(0) != '"') return "";
        return unescape(value.substring(1, value.length() - 1));
    }

    private static String rawValue(String json, String key) {
        String needle = "\"" + key + "\"";
        int at = json.indexOf(needle);
        if (at < 0) return null;
        int colon = json.indexOf(':', at + needle.length());
        if (colon < 0) return null;
        int i = colon + 1;
        while (i < json.length() && Character.isWhitespace(json.charAt(i))) i++;
        if (i >= json.length()) return null;
        char first = json.charAt(i);
        if (first == '"') {
            int end = i + 1;
            while (end < json.length()) {
                char c = json.charAt(end);
                if (c == '\\') {
                    end += 2;
                    continue;
                }
                if (c == '"') return json.substring(i, end + 1);
                end++;
            }
            return null;
        }
        if (first == '{' || first == '[') {
            int depth = 0;
            for (int end = i; end < json.length(); end++) {
                char c = json.charAt(end);
                if (c == '"') {
                    end = skipString(json, end);
                    continue;
                }
                if (c == '{' || c == '[') depth++;
                if (c == '}' || c == ']') {
                    depth--;
                    if (depth == 0) return json.substring(i, end + 1);
                }
            }
            return null;
        }
        int end = i;
        while (end < json.length() && json.charAt(end) != ',' && json.charAt(end) != '}') end++;
        return json.substring(i, end).trim();
    }

    private static int skipString(String json, int start) {
        for (int i = start + 1; i < json.length(); i++) {
            if (json.charAt(i) == '\\') {
                i++;
                continue;
            }
            if (json.charAt(i) == '"') return i;
        }
        return json.length() - 1;
    }

    private static String unescape(String value) {
        StringBuilder out = new StringBuilder(value.length());
        for (int i = 0; i < value.length(); i++) {
            char c = value.charAt(i);
            if (c != '\\' || i + 1 >= value.length()) {
                out.append(c);
                continue;
            }
            char next = value.charAt(++i);
            switch (next) {
                case 'n' -> out.append('\n');
                case 'r' -> out.append('\r');
                case 't' -> out.append('\t');
                case 'b' -> out.append('\b');
                case 'f' -> out.append('\f');
                case '"' -> out.append('"');
                case '\\' -> out.append('\\');
                case '/' -> out.append('/');
                case 'u' -> {
                    if (i + 4 < value.length()) {
                        out.append((char) Integer.parseInt(value.substring(i + 1, i + 5), 16));
                        i += 4;
                    }
                }
                default -> out.append(next);
            }
        }
        return out.toString();
    }

    private static String encode(String value) {
        return java.net.URLEncoder.encode(value, StandardCharsets.UTF_8);
    }
}
