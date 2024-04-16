function checkPassword() {
    const firstPasswordInput = document.getElementById('first-password')
    const confirmPasswordInput = document.getElementById('confirm-password')
    const to_add = confirmPasswordInput.value === firstPasswordInput.value ? 'border-green-500' : 'border-rose-700'
    const to_remove = confirmPasswordInput.value === firstPasswordInput.value ? 'border-rose-700' : 'border-green-500'
    confirmPasswordInput.classList.add(to_add)
    confirmPasswordInput.classList.remove(to_remove)
}


function create_user() {

    let fName = $('[name="fName"]').val();
    let lName = $('[name="lName"]').val();
    let email = $('[name="email"]').val();
    let contact = $('[name="contact"]').val();
    let gender = $("input[name='gender']:checked").val();
    let password = $('[name="password"]').val();
    let eFName = $('[name="eFName"]').val();
    let eLName = $('[name="eLName"]').val();
    let eContact = $('[name="eContact"]').val();
    let ePersonRelation = $("#ePersonRelation").val();
    let position = $("#position").val();

    if (fName === "" || lName === "" || email === "" || contact === "" || gender === "" || password === "" || eFName === "" || eLName === "" || eContact === "" || ePersonRelation === "" || position === "") {
        const Toast = Swal.mixin({
            toast: true,
            position: "top-end",
            showConfirmButton: false,
            timer: 3000,
            timerProgressBar: true,
            didOpen: (toast) => {
                toast.onmouseenter = Swal.stopTimer;
                toast.onmouseleave = Swal.resumeTimer;
            }
        });
        Toast.fire({
            icon: "error",
            title: "Please fill in all the fields!"
        });
        return;
    }

    console.log(fName, lName, email, contact, gender, password, eFName, eLName, eContact, ePersonRelation, position);

    $.ajax({
        contentType: "application/json",
        url: "/create_user_api",
        type: "POST",
        data: JSON.stringify({ "fName": fName, "lName": lName, "email": email, "contact": contact, "gender": gender, "password": password, "eFName": eFName, "eLName": eLName, "eContact": eContact, "ePersonRelation": ePersonRelation, "position": position }),
        success: function () {
            //clear all the fields
            // $('[name="fName"]').val('');
            // $('[name="lName"]').val('');
            // $('[name="email"]').val('');
            // $('[name="contact"]').val('');
            // $('[name="password"]').val('');
            // $('[name="eFName"]').val('');
            // $('[name="eLName"]').val('');
            // $('[name="eContact"]').val('');
            // $("#ePersonRelation").val('');
            // $("#postion").val('');


            const Toast = Swal.mixin({
                toast: true,
                position: "top-end",
                showConfirmButton: false,
                timer: 3000,
                timerProgressBar: true,
                didOpen: (toast) => {
                    toast.onmouseenter = Swal.stopTimer;
                    toast.onmouseleave = Swal.resumeTimer;
                }
            });
            Toast.fire({
                icon: "success",
                title: "User has been created successfully!"
            });
        },
        error: function () {
            const Toast = Swal.mixin({
                toast: true,
                position: "top-end",
                showConfirmButton: false,
                timer: 3000,
                timerProgressBar: true,
                didOpen: (toast) => {
                    toast.onmouseenter = Swal.stopTimer;
                    toast.onmouseleave = Swal.resumeTimer;
                }
            });
            Toast.fire({
                icon: "error",
                title: "User create failed!"
            });
        },
    })
}