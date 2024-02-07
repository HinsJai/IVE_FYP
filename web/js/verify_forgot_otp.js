$(document).ready(function () {
  $("#otp")
    .children()
    .keyup(function () {
      if ($(this).val().length == $(this).attr("maxlength")) {
        $(this).next().focus();
      }
    });
});

document.querySelector("button").addEventListener("click", (e) => {
  e.preventDefault();
  verify_otp();
});

function verify_otp() {
  var otp = "";
  $("#otp")
    .children()
    .each(function () {
      otp += $(this).val();
    });
  $("#otp_hidden").val(otp);
  $.ajax({
    contentType: "application/json",
    url: "/verify_otp_api",
    type: "POST",
    data: JSON.stringify({
      otp: parseInt(otp),
    }),
    success: function () {
      window.location.replace("./reset_password");
    },
    error: function () {
      alert("Invalid OTP");
    },
  });
}
