// Progressive enhancement only: the page is fully readable without this file.
(function () {
  var reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  var el = document.getElementById("typed");
  if (el && !reduce) {
    var lines;
    try { lines = JSON.parse(el.dataset.lines); } catch (e) { lines = null; }
    if (lines && lines.length) {
      var li = 0, ci = 0;
      var type = function () {
        var line = lines[li];
        el.textContent = line.slice(0, ++ci);
        if (ci < line.length) return setTimeout(type, 35);
        setTimeout(function () { li = (li + 1) % lines.length; ci = 0; type(); }, 1800);
      };
      el.textContent = "";
      type();
    }
  }

  if (!reduce && "IntersectionObserver" in window) {
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (e) {
        if (e.isIntersecting) { e.target.classList.add("in"); io.unobserve(e.target); }
      });
    }, { threshold: 0.1 });
    document.querySelectorAll(".card, .chips li, .stats div, .diagram").forEach(function (n) {
      n.classList.add("reveal");
      io.observe(n);
    });
  }
})();
