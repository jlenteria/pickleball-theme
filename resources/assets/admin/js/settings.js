'use strict';

// Stop video/iframe playback as soon as preview modals begin closing.
(function() {
  function stopPlayback(event) {
    var modal = event.target;
    
    var videos = modal.querySelectorAll('video');
    videos.forEach(function(video) {
      video.pause();
      try { video.currentTime = 0; } catch (ignore) {}
    });

    var iframes = modal.querySelectorAll('iframe');
    iframes.forEach(function(iframe) {
      var src = iframe.src;
      iframe.src = '';
      window.setTimeout(function() { iframe.src = src; }, 0);
    });
  }

  function setupPreviewModals() {
    var modals = document.querySelectorAll('.pickleball-preview-modal');

    modals.forEach(function(modal) {
      // Bootstrap emits both events; listening to hide stops playback before
      // the fade animation and hidden covers programmatic/late closures.
      modal.addEventListener('hide.bs.modal', stopPlayback);
      modal.addEventListener('hidden.bs.modal', stopPlayback);
    });
  }
  
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', setupPreviewModals);
  } else {
    setupPreviewModals();
  }
})();
