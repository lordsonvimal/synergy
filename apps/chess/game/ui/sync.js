import { ClientClock } from "./clock.js"

export const clocks = {
  white: new ClientClock(
    document.getElementById("white-clock")
  ),
  black: new ClientClock(
    document.getElementById("black-clock")
  )
}

export function onClockSync(msg) {
  clocks.white.sync(msg.white)
  clocks.black.sync(msg.black)
}
