'use strict';

// Lightweight local 3D rally game. The court is perspective-projected onto a
// canvas so it works offline without shipping a WebGL/game-engine dependency.
function setupPickleballGames() {
  document.querySelectorAll('[data-pickleball-game]').forEach(function (game) {
    var canvas = game.querySelector('[data-game-canvas]');
    var hit = game.querySelector('[data-game-hit]');
    var reset = game.querySelector('[data-game-reset]');
    var left = game.querySelector('[data-game-left]');
    var right = game.querySelector('[data-game-right]');
    var scoreEl = game.querySelector('[data-game-score]');
    var status = game.querySelector('[data-game-status]');
    var ctx = canvas.getContext('2d');
    if (!ctx) {
      status.textContent = 'This game is not supported by your browser.';
      return;
    }
    var target = Math.max(1, Number(game.dataset.targetScore) || 7);
    var difficulty = game.dataset.difficulty || 'medium';
    var travelMs = difficulty === 'hard' ? 1150 : difficulty === 'easy' ? 1800 : 1450;
    var timingWindow = difficulty === 'hard' ? .09 : difficulty === 'easy' ? .18 : .13;
    var player = 0, opponent = 0, playerX = 0, phase = 'ready';
    var ball = { x: 0, z: .84, y: .05, progress: 0 };
    var width = 0, height = 0, last = 0, raf;

    function resize() {
      var rect = canvas.getBoundingClientRect();
      var ratio = Math.min(window.devicePixelRatio || 1, 2);
      width = Math.max(280, rect.width); height = Math.max(180, rect.height);
      canvas.width = Math.round(width * ratio); canvas.height = Math.round(height * ratio);
      ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
    }
    function renderScore() { scoreEl.textContent = player + '–' + opponent; }
    function gameOver() {
      phase = 'over';
      status.textContent = player > opponent ? 'You win! Great rally.' : 'Game point to the opponent. Try again!';
    }
    function isWinner(a, b) { return a >= target && a - b >= 2; }
    function nextBall() {
      phase = 'incoming'; ball.x = Math.random() * 1.6 - .8; ball.progress = 0; ball.z = .84;
      status.textContent = 'Move under the ball, then tap Return.';
    }
    function point(won) {
      if (won) { player += 1; status.textContent = 'Clean return! Watch the next ball.'; }
      else { opponent += 1; status.textContent = 'Missed it—move closer next time.'; }
      renderScore();
      if (isWinner(player, opponent) || isWinner(opponent, player)) { gameOver(); return; }
      phase = 'between'; window.setTimeout(nextBall, 450);
    }
    function returnBall() {
      if (phase === 'ready' || phase === 'over') {
        if (phase === 'over') { player = 0; opponent = 0; renderScore(); }
        nextBall(); return;
      }
      if (phase !== 'incoming') return;
      var closeEnough = Math.abs(playerX - ball.x) < .32;
      var onTime = Math.abs(ball.progress - 1) <= timingWindow;
      point(closeEnough && onTime);
    }
    function move(amount) { playerX = Math.max(-.92, Math.min(.92, playerX + amount)); }
    function project(x, z, y) {
      var horizon = height * .28, depth = 1 - z, scale = .48 + depth * .72;
      return { x: width / 2 + x * width * .42 * scale, y: horizon + depth * height * .62 - y * height * .55 * scale, scale: scale };
    }
    function drawPaddle(point, color, angle) {
      ctx.save(); ctx.translate(point.x, point.y); ctx.rotate(angle || 0);
      ctx.fillStyle = color; ctx.shadowColor = 'rgba(0,0,0,.35)'; ctx.shadowBlur = 7;
      ctx.fillRect(-10, -25, 20, 37); ctx.shadowBlur = 0; ctx.fillRect(-3, 12, 6, 23); ctx.restore();
    }
    function draw() {
      ctx.clearRect(0, 0, width, height);
      var sky = ctx.createLinearGradient(0, 0, 0, height); sky.addColorStop(0, '#063739'); sky.addColorStop(1, '#0e7c7b');
      ctx.fillStyle = sky; ctx.fillRect(0, 0, width, height);
      var farL = project(-1, 0, 0), farR = project(1, 0, 0), nearR = project(1, 1, 0), nearL = project(-1, 1, 0);
      ctx.beginPath(); ctx.moveTo(farL.x, farL.y); ctx.lineTo(farR.x, farR.y); ctx.lineTo(nearR.x, nearR.y); ctx.lineTo(nearL.x, nearL.y); ctx.closePath();
      ctx.fillStyle = 'rgba(21,150,132,.78)'; ctx.fill(); ctx.strokeStyle = 'rgba(255,255,255,.9)'; ctx.lineWidth = 2; ctx.stroke();
      [.24, .5, .76].forEach(function (z) { var l = project(-1, z, 0), r = project(1, z, 0); ctx.beginPath(); ctx.moveTo(l.x, l.y); ctx.lineTo(r.x, r.y); ctx.strokeStyle = 'rgba(255,255,255,.42)'; ctx.stroke(); });
      var a = project(0, 0, 0), b = project(0, 1, 0); ctx.beginPath(); ctx.moveTo(a.x, a.y); ctx.lineTo(b.x, b.y); ctx.stroke();
      var netL = project(-1, .5, 0), netR = project(1, .5, 0); ctx.beginPath(); ctx.moveTo(netL.x, netL.y - 7); ctx.lineTo(netR.x, netR.y - 7); ctx.strokeStyle = 'rgba(255,255,255,.95)'; ctx.lineWidth = 4; ctx.stroke();
      drawPaddle(project(playerX, .92, .14), '#c6e92b', playerX * .2); drawPaddle(project(ball.x * .7, .05, .1), '#f4f7f1', 0);
      var bp = project(ball.x, ball.z, ball.y), radius = Math.max(4, 10 * bp.scale); ctx.beginPath(); ctx.arc(bp.x, bp.y, radius, 0, Math.PI * 2); ctx.fillStyle = '#c6e92b'; ctx.shadowColor = 'rgba(0,0,0,.45)'; ctx.shadowBlur = 8; ctx.fill(); ctx.shadowBlur = 0;
      ctx.fillStyle = 'rgba(255,255,255,.82)'; ctx.font = '11px system-ui'; ctx.textAlign = 'center'; ctx.fillText(phase === 'over' ? 'TAP NEW GAME' : phase === 'ready' ? 'TAP RETURN TO PLAY' : 'MOVE  •  RETURN', width / 2, height - 10);
    }
    function loop(now) {
      var delta = Math.min(50, now - last); last = now;
      if (phase === 'incoming') { ball.progress = Math.min(1, ball.progress + delta / travelMs); ball.z = .84 - ball.progress * .78; ball.y = .06 + Math.sin(ball.progress * Math.PI) * .42; if (ball.progress >= 1) point(false); }
      draw(); raf = window.requestAnimationFrame(loop);
    }
    function bindHold(button, amount) {
      var interval;
      function stop() { window.clearInterval(interval); interval = null; }
      button.addEventListener('pointerdown', function (event) { event.preventDefault(); move(amount); interval = window.setInterval(function () { move(amount); }, 70); });
      ['pointerup', 'pointercancel', 'pointerleave'].forEach(function (type) { button.addEventListener(type, stop); });
    }
    canvas.addEventListener('pointerdown', function (event) { var rect = canvas.getBoundingClientRect(); playerX = Math.max(-.92, Math.min(.92, ((event.clientX - rect.left) / rect.width - .5) * 2)); returnBall(); });
    canvas.addEventListener('keydown', function (event) { var key = event.key.toLowerCase(); if (key === 'arrowleft' || key === 'a') { event.preventDefault(); move(-.12); } if (key === 'arrowright' || key === 'd') { event.preventDefault(); move(.12); } if (key === ' ' || key === 'enter') { event.preventDefault(); returnBall(); } });
    hit.addEventListener('click', returnBall); reset.addEventListener('click', function () { player = 0; opponent = 0; playerX = 0; phase = 'ready'; renderScore(); status.textContent = 'Press Return to serve.'; });
    bindHold(left, -.12); bindHold(right, .12); window.addEventListener('resize', resize); resize(); renderScore(); raf = window.requestAnimationFrame(loop);
  });
}

if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', setupPickleballGames); else setupPickleballGames();

// Bootstrap modals must escape the portal shell's stacking context.
function liftCourtRatesModal() { document.querySelectorAll('.court-rates-modal').forEach(function (modal) { if (modal.parentNode !== document.body) document.body.appendChild(modal); }); }
if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', liftCourtRatesModal); else liftCourtRatesModal();

function meterHTML(text) {
  var html = '', run = '', runIsUnit = null;
  function flush() { if (run === '') return; html += runIsUnit ? '<i class="meter-unit">' + run + '</i>' : '<span class="meter-num">' + run + '</span>'; run = ''; }
  for (var i = 0; i < text.length; i++) { var ch = text.charAt(i); if (ch === ' ') continue; var isUnit = ch < '0' || ch > '9'; if (runIsUnit !== null && isUnit !== runIsUnit) flush(); runIsUnit = isUnit; run += ch; }
  flush(); return html;
}
function tick(el) { el.classList.remove('pickleball-tick'); void el.offsetWidth; el.classList.add('pickleball-tick'); }
function queuedExtra(el, kind) { var value = Number(el.getAttribute(kind === 'data' ? 'data-extra-data-mb' : 'data-extra-time-secs')); return isFinite(value) && value > 0 ? value : 0; }
window.$flare.ui.liveSession.setRenderer(function (el, text, value, kind) {
  if (kind === 'time' || kind === 'data') { var extra = queuedExtra(el, kind); if (extra > 0) { var total = value + extra; text = kind === 'data' ? window.$flare.ui.liveSession.formatByteData(total) : window.$flare.ui.liveSession.formatTimeSecs(total); } }
  el.innerHTML = meterHTML(text); tick(el);
});

document.addEventListener('keydown', function (event) {
  if (event.key !== 'Enter' && event.which !== 13) return;
  var target = event.target;
  if (!target || !target.closest || !target.closest('#login-form input') || !target.form) return;
  event.preventDefault();
  if (typeof target.form.requestSubmit === 'function') target.form.requestSubmit(); else target.form.submit();
});
