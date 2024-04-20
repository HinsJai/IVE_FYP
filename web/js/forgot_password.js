document.querySelector("#forget_password").addEventListener("submit", (e) => {
  e.preventDefault();
  verify_email();
});

function verify_email() {
  var email = $("input[name='email']").val();
  $.ajax({
    contentType: "application/json",
    type: "POST",
    url: "/verify_email_api",
    data: JSON.stringify({
      email: email,
    }),
    success: function () {
      window.location.replace("/verify_forgot_otp");
    },
    error: function () {
      alert("Email not found");
    },
  });
}

function blackToLogin() {
  window.location.replace("/");
}
