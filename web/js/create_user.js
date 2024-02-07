function checkPassword() {
    const firstPasswordInput = document.getElementById('first-password')
    const confirmPasswordInput = document.getElementById('confirm-password')
    const to_add = confirmPasswordInput.value === firstPasswordInput.value ? 'border-green-500' : 'border-rose-700'
    const to_remove = confirmPasswordInput.value === firstPasswordInput.value ? 'border-rose-700' : 'border-green-500'
    confirmPasswordInput.classList.add(to_add)
    confirmPasswordInput.classList.remove(to_remove)
}