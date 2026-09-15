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
})();
