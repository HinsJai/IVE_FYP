const errorMsg = document.getElementById("login-err");
window.onload = function () {
  if (errorMsg.innerHTML.trim() !== "") {
    errorMsg.style.display = "block";
    errorMsg.classList.add("bg-rose-800");
  }
};
errorMsg.addEventListener("click", function () {
  this.style.display = "none";
  window.location.replace("/");
});

