document.querySelector("button").addEventListener("click", (e) => {
  e.preventDefault();
  reset_passwpord();
});

function reset_passwpord() {
  $.ajax({
    contentType: "application/json",
    url: "/reset_password_api",
    type: "POST",
    data: JSON.stringify({
      password: $("input[name=password]").val(),
    }),
    success: function () {
      window.location.replace("./");
    },
    error: function () {
      alert("Rest password failed!");
    },
  });
}
