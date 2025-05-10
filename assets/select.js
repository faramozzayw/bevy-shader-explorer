const select = document.querySelector(".version-select");
const current = "/" + location.pathname.split("/")[1];
for (const option of select.options) {
  if (option.value === current) {
    option.selected = true;
    break;
  }
}
