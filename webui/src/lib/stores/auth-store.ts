import { writable } from 'svelte/store';
import { jwtDecode } from 'jwt-decode';

export type RefreshResultType = {
    errorCode: number
    errorMsg: string
};

export type UserData = {
    exp: number;
    access_uuid: string;
    username: string;
    role: string;
};

let accessToken = "";

const refreshResultValue = writable<RefreshResultType>({ errorCode: 0, errorMsg: "" });

const doneFirstAttemptValue = writable<boolean>(false);

const noAdminsRegisteredValue = writable<boolean>(false);

const loggedInValue = writable<boolean>(false, () => {
    doRefresh();
    const interval = setInterval(() => {
        doRefresh();
    }, 60000);
    return () => clearInterval(interval);
});

const userDataValue = writable<UserData>({
    exp: 0,
    access_uuid: "",
    username: "",
    role: ""
});

function setAccessToken(token: string) {
    accessToken = token;

    if (token !== "") {
        const claims = jwtDecode<UserData>(token);
        userDataValue.set(claims);
    } else {
        userDataValue.set({
            exp: 0,
            access_uuid: "",
            username: "",
            role: ""
        });
    }
}

export function getAccessToken() : string {
    return accessToken;
}

export const doneFirstAuthAttempt : { subscribe: (subscription: (value: boolean) => void) => (() => void) } = {
    subscribe: doneFirstAttemptValue.subscribe
};

export const noAdminsRegistered : { subscribe: (subscription: (value: boolean) => void) => (() => void) } = {
    subscribe: noAdminsRegisteredValue.subscribe
};

export const loggedIn : { subscribe: (subscription: (value: boolean) => void) => (() => void) } = {
    subscribe: loggedInValue.subscribe
};

export const userData = {
    subscribe: userDataValue.subscribe
};

export async function doLogin(username: string, password: string): Promise<RefreshResultType> {
    const response = await fetch("/api/login", {
        method: "POST",
        headers: { 'Content-Type': 'application/json' },
        body: '{"username":"' + username + '","password":"' + password + '"}'
    })
    // codes:
    //  500: Internal server error (vite serve could not proxy request)
    //  400: Bad request (incorrect username/password)
    if (response.ok) {
        try {
            const data = await response.json();
            setAccessToken(data.accessToken);
            // console.log("doLogin:", data);
            loggedInValue.set(accessToken != "");
            refreshResultValue.set({ errorCode: 0, errorMsg: "" });
            return { errorCode: 0, errorMsg: "" };
        } catch (error: any) {
            console.log("doLogin failed to parse json:", error);
            refreshResultValue.set({ errorCode: 1, errorMsg: error });
            return { errorCode: 1, errorMsg: error };
        }
    } else {
        const msg = await response.text();
        console.log("doLogin failed:", response.status, msg)
        refreshResultValue.set({ errorCode: response.status, errorMsg: msg });
        return { errorCode: response.status, errorMsg: msg };
    }
}

export async function doLogout() {
    const response = await fetch("/api/logout", {
        method: "POST",
        headers: { 'Content-Type': 'application/json' },
        body: '{}'
    })
    // codes:
    //  500: Internal server error (yarn serve could not proxy request)
    //  401: Unauthorized (token not provided)
    if (response.ok) {
        try {
            const data = await response.json();
            setAccessToken("");
            // console.log("doLogout:", data);
            loggedInValue.set(accessToken != "");
            refreshResultValue.set({ errorCode: 0, errorMsg: "" });
        } catch (error: any) {
            console.log("doLogout failed to parse json:", error);
            refreshResultValue.set({ errorCode: 1, errorMsg: error });
        }
    } else {
        const msg = await response.text();
        console.log("doLogout failed:", response.status, msg)
        refreshResultValue.set({ errorCode: response.status, errorMsg: msg });
    }
}

export async function doRefresh() {
    const response = await fetch("/api/refresh", {
        method: "POST",
        headers: { 'Content-Type': 'application/json' },
        body: '{}'
    })
    // codes:
    //  500: Internal server error (yarn serve could not proxy request)
    //  401: Unauthorized (token not provided)
    if (response.ok) {
        try {
            const data = await response.json();
            setAccessToken(data.accessToken);
            // console.log("doRefresh:", data);
            loggedInValue.set(accessToken != "");
            // doneFirstAttemptValue.set(true);
            refreshResultValue.set({ errorCode: 0, errorMsg: "" });
        } catch (error: any) {
            console.error("doRefresh failed to parse json:", error);
            // doneFirstAttemptValue.set(false);
            refreshResultValue.set({ errorCode: 1, errorMsg: error });
        }
    } else {
        const msg = await response.text();
        console.error("doRefresh response failed:", response.status, msg);
        if (response.status === 412) { // Precondition Failed
            noAdminsRegisteredValue.set(true);
        } else {
            noAdminsRegisteredValue.set(false);
        }
        // doneFirstAttemptValue.set(false);
        refreshResultValue.set({ errorCode: response.status, errorMsg: msg });
    }

    doneFirstAttemptValue.set(true);
}

export const AuthStore = {
    // refreshResult: { subscribe: refreshResultValue.subscribe },
    loggedIn,
    doneFirstAuthAttempt,
    noAdminsRegisteredValue,
    // doLogin,
    // doLogout,
    // renewAccessToken,
};
