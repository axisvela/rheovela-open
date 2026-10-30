import asyncio
import json
import tempfile
import unittest
from urllib.parse import parse_qs, urlsplit

try:
    import websockets
except ImportError:  # pragma: no cover
    websockets = None

from events import DurableQueue, Event, EventClient


class QueueTests(unittest.TestCase):
    def test_persist_and_ack(self):
        with tempfile.TemporaryDirectory() as directory:
            queue = DurableQueue(directory)
            queue.push(Event(seq=2, type="b"))
            queue.push(Event(seq=1, type="a"))
            queue.push(Event(seq=2, type="b"))

            reopened = DurableQueue(directory)
            self.assertEqual([1, 2], [item.seq for item in reopened.pending()])

            reopened.ack(1)
            self.assertEqual(1, reopened.last_seq)
            self.assertEqual([2], [item.seq for item in reopened.pending()])

            again = DurableQueue(directory)
            self.assertEqual(1, again.last_seq)
            self.assertEqual([2], [item.seq for item in again.pending()])


async def wait_until(predicate, timeout=3.0):
    deadline = asyncio.get_event_loop().time() + timeout
    while asyncio.get_event_loop().time() < deadline:
        if predicate():
            return
        await asyncio.sleep(0.01)
    raise AssertionError("condition not met before timeout")


@unittest.skipUnless(websockets, "websockets package is required")
class ClientTests(unittest.IsolatedAsyncioTestCase):
    async def test_receives_and_persists(self):
        async def handler(socket):
            await socket.send(json.dumps({"seq": 1, "type": "a"}))
            await socket.send(json.dumps({"seq": 2, "type": "b"}))
            await asyncio.sleep(10)

        server = await websockets.serve(handler, "127.0.0.1", 0)
        port = server.sockets[0].getsockname()[1]
        received = []
        with tempfile.TemporaryDirectory() as directory:
            client = EventClient(
                f"http://127.0.0.1:{port}", "S", directory,
                received.append,
                retry_delay=0.01, reconnect_min=0.01,
            )
            task = asyncio.create_task(client.run())
            await wait_until(lambda: len(received) >= 2)
            task.cancel()
            await asyncio.gather(task, return_exceptions=True)
            self.assertEqual([1, 2], [item.seq for item in received])
            self.assertEqual(2, client.last_seq)
            self.assertEqual([], client.pending())
        server.close()
        await server.wait_closed()

    async def test_reconnects_with_last_seq(self):
        after_seqs = []

        async def handler(socket):
            path = urlsplit(socket.request.path)
            after_seqs.append(parse_qs(path.query).get("after_seq", [""])[0])
            if len(after_seqs) == 1:
                await socket.send(json.dumps({"seq": 1, "type": "a"}))
                return
            await socket.send(json.dumps({"seq": 2, "type": "b"}))
            await asyncio.sleep(10)

        server = await websockets.serve(handler, "127.0.0.1", 0)
        port = server.sockets[0].getsockname()[1]
        received = []
        with tempfile.TemporaryDirectory() as directory:
            client = EventClient(
                f"http://127.0.0.1:{port}", "S", directory,
                received.append,
                retry_delay=0.01, reconnect_min=0.01,
            )
            task = asyncio.create_task(client.run())
            await wait_until(lambda: len(received) >= 2)
            task.cancel()
            await asyncio.gather(task, return_exceptions=True)
            self.assertEqual([1, 2], [item.seq for item in received])
            self.assertGreaterEqual(len(after_seqs), 2)
            self.assertEqual("", after_seqs[0])
            self.assertEqual("1", after_seqs[1])
        server.close()
        await server.wait_closed()

    async def test_handler_failure_retries(self):
        async def handler(socket):
            await socket.send(json.dumps({"seq": 1, "type": "a"}))
            await asyncio.sleep(10)

        server = await websockets.serve(handler, "127.0.0.1", 0)
        port = server.sockets[0].getsockname()[1]
        calls = []
        errors = []

        def consume(event):
            calls.append(event.seq)
            if len(calls) == 1:
                raise RuntimeError("transient")

        with tempfile.TemporaryDirectory() as directory:
            client = EventClient(
                f"http://127.0.0.1:{port}", "S", directory, consume,
                retry_delay=0.01, reconnect_min=0.01, on_error=errors.append,
            )
            task = asyncio.create_task(client.run())
            await wait_until(lambda: len(calls) >= 2 and client.last_seq == 1)
            task.cancel()
            await asyncio.gather(task, return_exceptions=True)
            self.assertGreaterEqual(len(calls), 2)
            self.assertTrue(errors)
            self.assertEqual([], client.pending())
        server.close()
        await server.wait_closed()


if __name__ == "__main__":
    unittest.main()
