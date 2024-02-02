function logout() {
    window.location.replace('/logout');
}
const dropdown = () => {
    document.querySelector('#drop-down-area').classList.toggle('hidden')
    document.querySelector('#drop-down-icon').classList.toggle('rotate-180')
}


