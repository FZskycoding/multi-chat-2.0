import ws from "k6/ws";
import { check } from "k6";
import { SharedArray } from "k6/data";

const scenarioPath =
  __ENV.SCENARIO_FILE || "../loadtest/minimal-scenario.json";
const scenario = new SharedArray("loadtest-scenario", function () {
  const raw = open(scenarioPath).replace(/^\uFEFF/, "");
  return [JSON.parse(raw)];
})[0];

const users = scenario.users || [];
const roomId = __ENV.ROOM_ID || scenario.room.id;
const baseUrl = __ENV.BASE_URL || scenario.baseUrl || "http://localhost:8080";
const wsUrl = baseUrl.replace(/^http/i, "ws") + "/ws";
const messageIntervalMs = Number(__ENV.MESSAGE_INTERVAL_MS || 5000); //每 5 秒發一次訊息
const waitForStartTimeoutMs = Number(__ENV.WAIT_FOR_START_TIMEOUT_MS || 60000);
const sendDurationMs = Number(__ENV.SEND_DURATION_MS || __ENV.CONNECTION_HOLD_MS || 60000); //收到開始訊號後持續送60秒
const virtualUsers = Number(__ENV.VUS || users.length || 1);
const scenarioMaxDurationSeconds =
  Math.ceil((waitForStartTimeoutMs + sendDurationMs) / 1000) + 30;

export const options = {
  scenarios: {
    websocket_chat_burst: {
      executor: "per-vu-iterations",
      vus: virtualUsers,
      iterations: 1,
      maxDuration: __ENV.MAX_DURATION || `${scenarioMaxDurationSeconds}s`,
    },
  },
};

if (users.length === 0) {
  throw new Error(
    `No users found in scenario file: ${scenarioPath}. Run the minimal scenario setup first.`
  );
}

if (virtualUsers > users.length) {
  throw new Error(
    `VUS=${virtualUsers} exceeds available tokens (${users.length}). Add more users or lower VUS.`
  );
}

export default function () {
  const user = users[__VU - 1];
  const params = { headers: { Cookie: `token=${user.token}` } };

  const res = ws.connect(wsUrl, params, function (socket) {
    let intervalHandle = null;
    let closeHandle = null;
    let startTimeoutHandle = null;
    let started = false;
    let loggedFirstMessage = false;
    let closed = false;

    const stopSession = () => {
      if (closed) {
        return;
      }
      closed = true;
      socket.close();
    };

    const startSending = () => {
      if (started) {
        return;
      }
      started = true;

      console.log(`VU ${__VU} received coordinated start signal`);

      intervalHandle = socket.setInterval(() => {
        const msg = JSON.stringify({
          type: "normal",
          roomId,
          content: `load test message from ${user.username} at ${new Date().toISOString()}`,
        });
        socket.send(msg);
      }, messageIntervalMs);

      closeHandle = socket.setTimeout(() => {
        console.log(`VU ${__VU} simulation finished, closing...`);
        stopSession();
      }, sendDurationMs);
    };

    socket.on("message", (data) => {
      if (!loggedFirstMessage) {
        console.log(`VU ${__VU} first incoming message raw: ${data}`);
        loggedFirstMessage = true;
      }

      let message;
      try {
        const text = typeof data === "string" ? data : `${data}`;
        message = JSON.parse(text);
      } catch (error) {
        console.log(`VU ${__VU} failed to parse incoming message: ${error}`);
        return;
      }

      console.log(
        `VU ${__VU} parsed incoming message type=${message.type} content=${message.content}`
      );

      if (
        message &&
        (message.type === "loadtest_start" || message.content === "start")
      ) {
        try {
          startSending();
        } catch (error) {
          console.log(`VU ${__VU} failed to start sending: ${error}`);
        }
      }
    });

    socket.on("open", () => {
      console.log(`VU ${__VU} connected as ${user.username}`);

      startTimeoutHandle = socket.setTimeout(() => {
        if (!started) {
          console.log(`VU ${__VU} did not receive coordinated start signal in time`);
          stopSession();
        }
      }, waitForStartTimeoutMs);
    });

    socket.on("close", () => console.log(`VU ${__VU} disconnected`));

    socket.on("error", (e) => {
      if (e.error() !== "websocket: close 1000 (normal)") {
        console.log(`VU ${__VU} error: ${e.error()}`);
      }
    });
  });

  check(res, { "status is 101": (r) => r && r.status === 101 });
}
