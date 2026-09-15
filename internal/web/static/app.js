(() => {
  const locate = document.querySelector('#locate');
  const status = document.querySelector('#location-status');
  if (locate) locate.addEventListener('click', () => {
    if (!navigator.geolocation) { status.textContent = 'Location is unavailable in this browser. Enter a place or pincode.'; return; }
    locate.disabled = true; status.textContent = 'Waiting for location permission…';
    navigator.geolocation.getCurrentPosition((position) => {
      document.querySelector('#latitude').value = position.coords.latitude.toFixed(6);
      document.querySelector('#longitude').value = position.coords.longitude.toFixed(6);
      status.textContent = `Location captured to about ${Math.round(position.coords.accuracy)} m. Confirm the place name above.`;
      locate.disabled = false;
    }, () => { status.textContent = 'Location permission was not granted. Enter a place or pincode.'; locate.disabled = false; }, {enableHighAccuracy: false, timeout: 10000, maximumAge: 300000});
  });
  const range = document.querySelector('#day-use'), out = document.querySelector('#day-output');
  if (range) range.addEventListener('input', () => out.textContent = `${range.value}%`);

  const assessment = document.querySelector('form.assessment');
  if (assessment) {
    try {
      const saved = JSON.parse(sessionStorage.getItem('solarsense-assessment') || '{}');
      Object.entries(saved).forEach(([name, value]) => {
        const fields = assessment.elements.namedItem(name);
        if (!fields || name === 'csrf_token') return;
        if (fields instanceof RadioNodeList) Array.from(fields).forEach((field) => { field.checked = field.value === value; });
        else if (fields.type !== 'file') fields.value = value;
      });
      if (saved.day_use && out) out.textContent = `${saved.day_use}%`;
    } catch (_) { sessionStorage.removeItem('solarsense-assessment'); }

    document.querySelectorAll('a[href="/auth/google"]').forEach((link) => link.addEventListener('click', () => {
      const values = {};
      new FormData(assessment).forEach((value, key) => { if (typeof value === 'string' && key !== 'csrf_token') values[key] = value; });
      sessionStorage.setItem('solarsense-assessment', JSON.stringify(values));
    }));
  }
})();
