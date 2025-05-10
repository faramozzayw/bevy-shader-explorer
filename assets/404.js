const messages = [
  "Maybe it got lost in the render graph? 🧭",
  "Did you check the visibility group? It might be culled 👀",
  "This shader took an unexpected branch and never returned 🌲",
  "Looks like it desynchronized from reality (or the GPU) 🔌",
  "404: Shader slipped through a missing bind group 🕳️",
  "Pipeline stage missing. Try re-binding your expectations 🔄",
  "This page was optimized out by the compiler 🗑️",
  "Maybe it’s hiding behind a conditional discard 🎭",
];

const msgElement = document.getElementById("message");
const randomIndex = Math.floor(Math.random() * messages.length);
msgElement.textContent = messages[randomIndex];
