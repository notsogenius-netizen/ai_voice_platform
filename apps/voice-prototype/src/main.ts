import {
  ConnectionState,
  Room,
  RoomEvent,
  Track,
  createLocalAudioTrack,
} from "livekit-client";

type SessionResponse = {
  room: string;
  livekit_url: string;
  token: string;
  identity: string;
};

const gatewayBase =
  import.meta.env.VITE_VOICE_GATEWAY_URL ?? "http://127.0.0.1:8080";

const connectBtn = document.querySelector<HTMLButtonElement>("#connectBtn")!;
const disconnectBtn =
  document.querySelector<HTMLButtonElement>("#disconnectBtn")!;
const connectionStateEl = document.querySelector("#connectionState")!;
const roomNameEl = document.querySelector("#roomName")!;
const identityEl = document.querySelector("#identity")!;
const micStateEl = document.querySelector("#micState")!;
const participantsEl = document.querySelector("#participants")!;
const logEl = document.querySelector("#log")!;

let room: Room | null = null;

function log(message: string): void {
  const line = `[${new Date().toLocaleTimeString()}] ${message}`;
  logEl.textContent = `${line}\n${logEl.textContent ?? ""}`;
}

function setConnectionState(state: string): void {
  connectionStateEl.textContent = state;
}

function refreshParticipants(active: Room): void {
  const names = [active.localParticipant.identity];
  active.remoteParticipants.forEach((p) => names.push(p.identity));
  participantsEl.textContent = names.join(", ") || "—";
}

function wireRoomEvents(active: Room): void {
  active.on(RoomEvent.ConnectionStateChanged, (state: ConnectionState) => {
    setConnectionState(state);
    log(`connection: ${state}`);
  });
  active.on(RoomEvent.ParticipantConnected, (p) => {
    log(`participant connected: ${p.identity}`);
    refreshParticipants(active);
  });
  active.on(RoomEvent.ParticipantDisconnected, (p) => {
    log(`participant disconnected: ${p.identity}`);
    refreshParticipants(active);
  });
  active.on(RoomEvent.LocalTrackPublished, (pub) => {
    if (pub.track?.kind === Track.Kind.Audio) {
      micStateEl.textContent = `published (${pub.trackName})`;
      log(`local mic published: ${pub.trackName}`);
    }
  });
}

async function createSession(): Promise<SessionResponse> {
  const res = await fetch(`${gatewayBase}/v1/sessions`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({}),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`session failed (${res.status}): ${text}`);
  }
  return (await res.json()) as SessionResponse;
}

async function connectAndPublish(): Promise<void> {
  connectBtn.disabled = true;
  setConnectionState("requesting session");
  log(`requesting session from ${gatewayBase}`);

  const session = await createSession();
  roomNameEl.textContent = session.room;
  identityEl.textContent = session.identity;
  log(`session room=${session.room} identity=${session.identity}`);

  const next = new Room();
  wireRoomEvents(next);
  room = next;

  setConnectionState("connecting");
  await next.connect(session.livekit_url, session.token);
  log(`connected to ${session.livekit_url}`);

  const mic = await createLocalAudioTrack();
  await next.localParticipant.publishTrack(mic);
  refreshParticipants(next);

  disconnectBtn.disabled = false;
}

async function disconnect(): Promise<void> {
  disconnectBtn.disabled = true;
  if (room) {
    await room.disconnect();
    room = null;
  }
  setConnectionState("disconnected");
  micStateEl.textContent = "not published";
  participantsEl.textContent = "—";
  connectBtn.disabled = false;
  log("disconnected");
}

connectBtn.addEventListener("click", () => {
  connectAndPublish().catch((err: unknown) => {
    const message = err instanceof Error ? err.message : String(err);
    setConnectionState("error");
    log(`error: ${message}`);
    connectBtn.disabled = false;
    disconnectBtn.disabled = !room;
  });
});

disconnectBtn.addEventListener("click", () => {
  disconnect().catch((err: unknown) => {
    const message = err instanceof Error ? err.message : String(err);
    log(`disconnect error: ${message}`);
  });
});
