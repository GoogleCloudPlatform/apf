/**
 * Copyright 2026 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * Custom k6 WebSocket Client Wrapper
 * Includes custom metrics: ws_connecting_errors, ws_frame_latency_ms
 */

import ws from 'k6/ws';
import { Counter, Trend } from 'k6/metrics';

export const ws_connecting_errors = new Counter('ws_connecting_errors');
export const ws_frame_latency_ms = new Trend('ws_frame_latency_ms');

export function connectWebSocket(url, params = {}, sessionCallback) {
  const response = ws.connect(url, params, (socket) => {
    let sendTime = 0;

    socket.on('open', () => {
      if (sessionCallback) {
        sessionCallback(socket, {
          sendPing: () => {
            sendTime = Date.now();
            socket.ping();
          },
        });
      }
    });

    socket.on('pong', () => {
      if (sendTime > 0) {
        ws_frame_latency_ms.add(Date.now() - sendTime);
      }
    });

    socket.on('error', (err) => {
      ws_connecting_errors.add(1);
    });

    socket.on('close', () => {
      // Graceful disconnect handling
    });
  });

  if (!response || response.status !== 101) {
    ws_connecting_errors.add(1);
  }

  return response;
}

export default {
  ws_connecting_errors,
  ws_frame_latency_ms,
  connectWebSocket,
};
