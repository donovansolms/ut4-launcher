$(document).ready(function(){
  $.backstretch([
      "images/backgrounds/bg01.jpg",
      //"images/backgrounds/bg02.jpg",
      //"images/backgrounds/bg03.jpg",
      "images/backgrounds/bg04.jpg"
  ], {duration: 10000, fade: 3000});

  $('#content_pane').niceScroll({
    cursorborder: "none",
    cursorcolor: "#F1F1F1",
    cursorwidth: "10px",
     railpadding: { top: 4, right: 10, left: 0, bottom: 4 },
     cursoropacitymin: 0.2,
     cursoropacitymax: 0.5,
  });
});
