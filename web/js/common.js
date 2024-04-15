$(function () {
  var includes = $("[data-include]");
  $.each(includes, function () {
    var file = $(this).data("include") + ".html";
    $(this).load(file);
  });
});

function logout() {
  Swal.fire({
    title: 'Do you confirm logout?', showCancelButton: true, confirmButtonText: 'Yes',
  }).then((result) => {
    if (result.isConfirmed) {
      window.location.replace("/logout");
    }
  })
}
function dropdown() {
  document.querySelector("#drop-down-area").classList.toggle("hidden");
  document.querySelector("#drop-down-icon").classList.toggle("rotate-180");
}


function updateDateTime() {
  $("#date-time-now").text(new Date().toLocaleString());
}
setInterval(updateDateTime, 1000);
