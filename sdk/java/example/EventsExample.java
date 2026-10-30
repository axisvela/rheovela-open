package example;

import worker.Events;

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.InetAddress;
import java.net.ServerSocket;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.MessageDigest;
import java.util.ArrayList;
import java.util.Base64;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

/** Runnable verification for the Java durable event client. */
public final class EventsExample {

    private static final String GUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11";

    private EventsExample() {}

    interface ConnectionHandler {
        void handle(int index, String afterSeq, Socket socket, OutputStream out) throws Exception;
    }

    static ServerSocket serve(int connections, ConnectionHandler handler) throws Exception {
        ServerSocket server = new ServerSocket(0, 50, InetAddress.getByName("127.0.0.1"));
        Thread thread = new Thread(() -> {
            try {
                for (int index = 1; index <= connections; index++) {
                    Socket socket = server.accept();
                    BufferedReader reader = new BufferedReader(
                            new InputStreamReader(socket.getInputStream(), StandardCharsets.ISO_8859_1));
                    String requestLine = reader.readLine();
                    String line;
                    String key = "";
                    while ((line = reader.readLine()) != null && !line.isEmpty()) {
                        if (line.toLowerCase().startsWith("sec-websocket-key:")) {
                            key = line.substring(line.indexOf(':') + 1).trim();
                        }
                    }
                    String accept = Base64.getEncoder().encodeToString(MessageDigest.getInstance("SHA-1")
                            .digest((key + GUID).getBytes(StandardCharsets.ISO_8859_1)));
                    OutputStream out = socket.getOutputStream();
                    out.write(("HTTP/1.1 101 Switching Protocols\r\n"
                            + "Upgrade: websocket\r\n"
                            + "Connection: Upgrade\r\n"
                            + "Sec-WebSocket-Accept: " + accept + "\r\n\r\n")
                            .getBytes(StandardCharsets.ISO_8859_1));
                    out.flush();
                    String target = requestLine.split(" ")[1];
                    String query = target.contains("?") ? target.substring(target.indexOf('?') + 1) : "";
                    handler.handle(index, afterSeq(query), socket, out);
                }
            } catch (Exception ignored) {
                // server closes with the test
            }
        });
        thread.setDaemon(true);
        thread.start();
        return server;
    }

    static String afterSeq(String query) {
        for (String pair : query.split("&")) {
            if (pair.startsWith("after_seq=")) {
                return pair.substring("after_seq=".length());
            }
        }
        return "";
    }

    static void writeClose(OutputStream out) throws Exception {
        out.write(0x88);
        out.write(0);
        out.flush();
    }

    static void writeText(OutputStream out, String json) throws Exception {
        byte[] payload = json.getBytes(StandardCharsets.UTF_8);
        out.write(0x81);
        out.write(payload.length);
        out.write(payload);
        out.flush();
    }

    static void close(ServerSocket server) throws Exception {
        server.close();
    }

    public static void main(String[] args) throws Exception {
        queuePersists();
        receivesAndPersists();
        reconnectsWithLastSeq();
        retriesHandlerFailures();
        System.out.println("Java events example: OK");
    }

    static void queuePersists() throws Exception {
        Path dir = Files.createTempDirectory("rheovela-events");
        Events.DurableQueue queue = new Events.DurableQueue(dir);
        queue.push(Events.parse("{\"seq\":2,\"type\":\"b\"}"));
        queue.push(Events.parse("{\"seq\":1,\"type\":\"a\"}"));
        if (queue.pending().size() != 2 || queue.pending().get(0).seq() != 1) {
            throw new AssertionError("pending queue order");
        }
        Events.DurableQueue reopened = new Events.DurableQueue(dir);
        if (reopened.lastSeq() != 0 || reopened.pending().size() != 2) {
            throw new AssertionError("queue persistence");
        }
        reopened.ack(1);
        Events.DurableQueue again = new Events.DurableQueue(dir);
        if (again.lastSeq() != 1 || again.pending().size() != 1 || again.pending().get(0).seq() != 2) {
            throw new AssertionError("queue ack persistence");
        }
    }

    static void receivesAndPersists() throws Exception {
        ServerSocket server = serve(1, (index, afterSeq, socket, out) -> {
            writeText(out, "{\"seq\":1,\"type\":\"a\"}");
            writeText(out, "{\"seq\":2,\"type\":\"b\"}");
            Thread.sleep(10_000);
        });
        CountDownLatch latch = new CountDownLatch(2);
        List<Long> received = new ArrayList<>();
        Path dir = Files.createTempDirectory("rheovela-events");
        Events.Client client = new Events.Client("http://127.0.0.1:" + server.getLocalPort(), "S", dir,
                event -> {
                    synchronized (received) {
                        received.add(event.seq());
                    }
                    latch.countDown();
                }, List.of(), "", "", 0, 10, 1000, 10, null);
        Thread runner = new Thread(client::run);
        runner.setDaemon(true);
        runner.start();
        if (!latch.await(5, TimeUnit.SECONDS)) {
            throw new AssertionError("timed out waiting for events");
        }
        client.close();
        if (client.lastSeq() != 2 || !client.pending().isEmpty()) {
            throw new AssertionError("client cursor/pending");
        }
        close(server);
    }

    static void reconnectsWithLastSeq() throws Exception {
        List<String> afterSeqs = new ArrayList<>();
        AtomicInteger connections = new AtomicInteger();
        ServerSocket server = serve(2, (index, afterSeq, socket, out) -> {
            connections.incrementAndGet();
            synchronized (afterSeqs) {
                afterSeqs.add(afterSeq);
            }
            if (index == 1) {
                writeText(out, "{\"seq\":1,\"type\":\"a\"}");
                writeClose(out);
                socket.close();
                return;
            }
            writeText(out, "{\"seq\":2,\"type\":\"b\"}");
            Thread.sleep(10_000);
        });
        CountDownLatch latch = new CountDownLatch(2);
        Path dir = Files.createTempDirectory("rheovela-events");
        Events.Client client = new Events.Client("http://127.0.0.1:" + server.getLocalPort(), "S", dir,
                event -> latch.countDown(), List.of(), "", "", 0, 10, 1000, 10, null);
        Thread runner = new Thread(client::run);
        runner.setDaemon(true);
        runner.start();
        if (!latch.await(5, TimeUnit.SECONDS)) {
            throw new AssertionError("timed out waiting for reconnect");
        }
        client.close();
        synchronized (afterSeqs) {
            if (afterSeqs.size() < 2 || !afterSeqs.get(0).isEmpty() || !"1".equals(afterSeqs.get(1))) {
                throw new AssertionError("after_seq reconnect: " + afterSeqs);
            }
        }
        close(server);
    }

    static void retriesHandlerFailures() throws Exception {
        ServerSocket server = serve(1, (index, afterSeq, socket, out) -> {
            writeText(out, "{\"seq\":1,\"type\":\"a\"}");
            Thread.sleep(10_000);
        });
        AtomicInteger calls = new AtomicInteger();
        List<Exception> errors = new ArrayList<>();
        Path dir = Files.createTempDirectory("rheovela-events");
        Events.Client client = new Events.Client("http://127.0.0.1:" + server.getLocalPort(), "S", dir,
                event -> {
                    if (calls.incrementAndGet() == 1) {
                        throw new IllegalStateException("transient");
                    }
                }, List.of(), "", "", 0, 10, 1000, 10, errors::add);
        Thread runner = new Thread(client::run);
        runner.setDaemon(true);
        runner.start();
        long deadline = System.currentTimeMillis() + 5000;
        while (System.currentTimeMillis() < deadline) {
            if (calls.get() >= 2 && client.lastSeq() == 1 && client.pending().isEmpty()) {
                break;
            }
            Thread.sleep(10);
        }
        client.close();
        if (calls.get() < 2 || client.lastSeq() != 1 || errors.isEmpty()) {
            throw new AssertionError("handler retry: calls=" + calls.get() + " last=" + client.lastSeq());
        }
        close(server);
    }
}
