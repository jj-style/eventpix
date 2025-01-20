document.body.addEventListener('showErrorToast', function (e) {
    const toast = document.getElementById('errorToast');
    const toastBootstrap = bootstrap.Toast.getOrCreateInstance(toast);
    toastBootstrap.show()
})