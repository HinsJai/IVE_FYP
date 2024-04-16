$(document).ready(function () {
    $("#update").click(function () {
        setup_user_profile_setting();
    })
})

function loadSetting() {
    get_user_profile_setting().then((data) => {
        if (data) {
            for (let element of data) {
                $(`#btn-${element}`).prop("checked", true);
            };
        }
    });

    get_helment_roles().then((data) => {
        if (data) {
            for (const [key, value] of Object.entries(data)) {
       
                $(`#label-${key}`).innerHTML = value;
                $(`#role-${key}`).val(value);
            };
        }
    });
}

function cancel() {
    loadSetting();
}

async function get_helment_roles() {
    response = await fetch(`/get_helment_roles`)
    if (response.status === 401) {
        window.location.replace("/?unauth=true")
        throw new Error("not login")
    }
    data = await response.json()
    return data[0].result[0]["role"]
}

async function get_user_profile_setting() {
    response = await fetch(`/get_setting_profile_api`)
    if (response.status === 401) {
        window.location.replace("/?unauth=true")
        throw new Error("not login")
    }
    data = await response.json()
    return data[0].result[0]["profileSetting"]
}

function setup_user_profile_setting() {
    let profileSetting = [];
    $("input:checkbox[name=profileSetting]:checked").each(function () {
        profileSetting.push(parseInt($(this).val()));
    });

    $.ajax({
        contentType: "application/json",
        url: "/set_setting_profile_api",
        type: "POST",
        data: JSON.stringify({ "data": profileSetting }),

        success: function () {
            set_helment_role();
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
                title: "Setting update failed"
            });
        },
    });
}

function set_helment_role() {
    let helment_roles = {};
    for (let i = 10; i < 14; i++) {
        if ($(`#role-${i}`).val() == "") {
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
                title: "Please fill in all the fields"
            });
            return;
        }
        helment_roles[i] = $(`#role-${i}`).val();
    }

    $.ajax({
        contentType: "application/json",
        url: "/set_helment_role_api",
        type: "POST",
        data: JSON.stringify(helment_roles),

        success: function () {
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
                title: "Setting has been updated successfully"
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
                title: "Setting update failed"
            });
        },
    });

}

window.onload = get_user_profile_setting;
window.onload = loadSetting();
