function logout() {
  window.location.replace("/logout");
}
function dropdown() {
  document.querySelector("#drop-down-area").classList.toggle("hidden");
  document.querySelector("#drop-down-icon").classList.toggle("rotate-180");
}

$(function () {
  var includes = $("[data-include]");
  $.each(includes, function () {
    var file = $(this).data("include") + ".html";
    $(this).load(file);
  });
});
