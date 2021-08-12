function LoadFindAccount() {
    $('#thecontent').load('find-account.html');
}

function LoadCreateAccount() {
    $('#thecontent').load('create-account.html');
}

function LoadNewJournal() {
    $('#thecontent').load('new-journal.html');
}

function LoadOpenJournal() {
    $('#thecontent').load('open-journal.html');
}

function LoadNewCurrency() {
    $('#thecontent').load('new-currency.html');
}

function LoadListCurrency() {
    $('#thecontent').load('list-currency.html');
}

function LoadCurrencyExchange() {
    $('#thecontent').load('currency-exchange.html');
}

function CurrentRFC3339Date() {
    let currentdate = new Date();

    let ye = new Intl.DateTimeFormat('en', { year: 'numeric' }).format(currentdate);
    let mo = new Intl.DateTimeFormat('en', { month: '2-digit' }).format(currentdate);
    if (mo.length !== 2) {
        mo = "0" + mo;
    }
    let da = new Intl.DateTimeFormat('en', { day: '2-digit' }).format(currentdate);
    if (da.length !== 2) {
        da = "0" + da;
    }
    let hr = new Intl.DateTimeFormat('en', { hour: '2-digit', hour12: false }).format(currentdate);
    if (hr.length !== 2) {
        hr = "0" + hr;
    }
    let mi = new Intl.DateTimeFormat('en', { minute: '2-digit' }).format(currentdate);
    if (mi.length !== 2) {
        mi = "0" + mi;
    }
    let se = new Intl.DateTimeFormat('en', { second: '2-digit' }).format(currentdate);
    if (se.length !== 2) {
        se = "0" + se;
    }
    return `${ye}-${mo}-${da}T${hr}:${mi}:${se}Z`;
}

function GenHMAC() {
    let userSecret = $("#theSecretKey").val();
    let date = CurrentRFC3339Date();
    let newHMAC = CryptoJS.HmacSHA256(date, userSecret);
    let base64encoded = CryptoJS.enc.Base64.stringify(newHMAC);
    let toBase = `${date}$${base64encoded}`;
    return btoa(toBase);
}

function GetCurrencyList() {
    $.ajax({
        url: '/api/v1/currencies',
        headers: { 'Authorization': GenHMAC() },
        success: function (data) {
            if (data.status !== "SUCCESS") {
                $("#currencyListBody").html("<tr><th scope=\"row\">&nbsp;</th><td colspan='3'>NO CURRENCY FOUND</td></tr>");
            } else {
                let currencyListRecords = "";
                for (let i = 0; i < data.data.length; i++) {
                    currencyListRecords = currencyListRecords + "<tr><th scope=\"row\">"+ (i+1) +"</th><td>" + data.data[i].code + "</td><td>" + data.data[i].name + "</td><td>" + data.data[i].exchange + "</td></tr>";
                }
                $("#currencyListBody").html(currencyListRecords);
            }
        },
        error: function(data, errorThrown) {
            console.log("Call failed : http://localhost:50051/api/v1/currencies : " + errorThrown);
            $("#currencyListBody").html("<tr><th scope=\"row\">&nbsp;</th><td colspan='3'>ERROR : "+data.message+"</td></tr>");
        },
        statusCode: {
            400: function (xhr) {
                window.alert("Seems you're input is wrong, check for date formats or mandatory fields.");
            },
            401: function (xhr) {
                window.alert("Seems your secret key is wrong.");
            }
        }
    });
}

function CallFindAccount() {
    FindAccount($("#accountNameFindField").val(),  1, parseInt($('#accountListItemCount').find(":selected").text()));
}

function LoadAccountDetail(accNo) {
    $('#thecontent').load('open-account-detail.html');

    $.ajax({
        url: "/api/v1/accounts/" + accNo,
        headers: { 'Authorization': GenHMAC() },
        success: function (data) {
            if (data.status !== "SUCCESS") {
                console.error("Error : " + data.message)
            } else {
                $('#accountInfoLabel').html("No : " + data.data.account_number);
                $('#accountNumberFindField').val(data.data.account_number);
                $('#accountNameFindField').val(data.data.name);
                $('#accountDescriptionField').val(data.data.description);
                $('#accountCOA').val(data.data.coa);
                $('#accountCurrencyField').val(data.data.currency);
                $('#accountAlignmentField').val(data.data.alignment);
                $('#accountBalanceField').val(data.data.balance);
            }
        },
        error: function(data, errorThrown) {
            console.error("Error : " + errorThrown)
        },
        statusCode: {
            400: function (xhr) {
                window.alert("Seems you're input is wrong, check for date formats or mandatory fields.");
            },
            401: function (xhr) {
                window.alert("Seems your secret key is wrong.");
            }
        }
    });
}

function LoadAccountTransactions(pageNo, items) {
    let dateFrom = $('#transactionTimeFrom').val();
    let dateUntil = $('#transactionTimeUntil').val();
    if (dateFrom.length === 0 || dateUntil.length === 0) {
        window.alert("From and Until field must not be empty");
        return;
    }
    let accNo = $('#accountNumberFindField').val();
    let from = dateFrom+"T00:00:00";
    let until = dateUntil+"T23:59:59";

    $.ajax({
        url: "/api/v1/accounts/" + accNo + "/transactions?from="+from+"&until="+until+"&page="+pageNo+"&size=" + items,
        headers: { 'Authorization': GenHMAC() },
        success: function (data) {
            if (data.status !== "SUCCESS") {
                console.error("Error : " + data.message)
            } else {
                let transactionListRecords = "";
                for (let i = 0; i < data.data.transactions.length; i++) {
                    let trx = data.data.transactions[i];
                    transactionListRecords = transactionListRecords +
                        " <tr><th scope=\"row\">"+trx.transaction_time+"</th>" +
                        "<td>"+trx.journal_id+"</td>" +
                        "<td>"+trx.transaction_id+"</td>" +
                        "<td>"+trx.description+"</td>";
                    if (trx.transaction_type === "DEBIT") {
                        transactionListRecords = transactionListRecords +
                            "<td>"+trx.amount+"</td>" +
                            "<td>&nbsp;</td>";
                    } else {
                        transactionListRecords = transactionListRecords +
                            "<td>&nbsp;</td>" +
                            "<td>"+trx.amount+"</td>" ;
                    }
                    transactionListRecords = transactionListRecords +
                        "<td>"+trx.account_balance+"</td>" +
                        "<td><a href=\"#\" class=\"btn btn-info btn-circle btn-sm\">" +
                        "<i class=\"fas fa-info-circle\"></i>" +
                        "</a></td></tr>";
                }
                $('#transactionListRows').html(transactionListRecords)
            }
        },
        error: function(data, errorThrown) {
            console.error("Error : " + errorThrown)
        },
        statusCode: {
            400: function (xhr) {
                window.alert("Seems you're input is wrong, check for date formats or mandatory fields.");
            },
            401: function (xhr) {
                window.alert("Seems your secret key is wrong.");
            }
        }
    });
}

function OpenJournal() {
    let journalNo = $('#journalNumberField').val();
    if (journalNo.length === 0) {
        window.alert("You need to specify the journal number to load");
        return;
    }
    $.ajax({
        url: "/api/v1/journals/"+journalNo,
        headers: { 'Authorization': GenHMAC() },
        success: function (data) {
            if (data.status !== "SUCCESS") {
                $("#findAccountRows").html("<tr><th scope=\"row\">&nbsp;</th><td colspan='6'>NO ACCOUNT WITH THAT CRITERIA IS FOUND</td></tr>");
            } else {
                $("#journalIdField").val(data.data.journal_id);
                $("#journalTimeField").val(data.data.journaling_time);
                $("#journalAmountField").val(data.data.amount);
                $("#journalDescriptionField").val(data.data.description);
                if (data.data.reversal) {
                    $("#journalIsReversalField").val("Yes");
                    $("#reversedJournalField").val(data.data.reversed_journal);
                } else {
                    $("#journalIsReversalField").val("No");
                    $("#reversedJournalField").val("N/A");
                }

                let transactionListRecords = "";
                let totalCredit = 0;
                let totalDebit = 0;
                for (let i = 0; i < data.data.Transactions.length; i++) {
                    let trx = data.data.Transactions[i];
                    if (trx.transaction_type === "DEBIT") {
                        transactionListRecords = transactionListRecords + "<tr>" +
                        "<td>"+trx.transaction_id+"</td>" +
                        "<td>"+trx.account_number+"</td>" +
                        "<td>"+trx.description+"</td>" +
                        "<td>" + trx.amount + "</td>" +
                        "<td>&nbsp;</td></tr>";
                        totalDebit += trx.amount;
                    }
                }
                for (let i = 0; i < data.data.Transactions.length; i++) {
                    let trx = data.data.Transactions[i];
                    if (trx.transaction_type === "CREDIT") {
                    transactionListRecords = transactionListRecords + "<tr>" +
                        "<td>"+trx.transaction_id+"</td>" +
                        "<td>"+trx.account_number+"</td>" +
                        "<td>"+trx.description+"</td>" +
                        "<td>&nbsp;</td>" +
                        "<td>" + trx.amount + "</td></tr>";
                        totalCredit += trx.amount;
                    }
                }
                $("#transactionAccountRows").html(transactionListRecords);
                $("#totalDebit").text(totalDebit);
                $("#totalCredit").text(totalCredit);
            }
        },
        error: function(data, errorThrown) {
            $("#findAccountRows").html("<tr><th scope=\"row\">&nbsp;</th><td colspan='6'>ERROR : "+errorThrown+"</td></tr>");
        },
        statusCode: {
            404: function (xhr) {
                window.alert("Journal Number " + journalNo + " is not exist")
            },
            400: function (xhr) {
                window.alert("Seems you're input is wrong, check for date formats or mandatory fields.");
            },
            401: function (xhr) {
                window.alert("Seems your secret key is wrong.");
            }
        }
    });
}

function FindAccount(name, page, items) {
    if (name.length < 3) {
        window.alert("Account Name is less than 3 characters")
        return
    }
    $.ajax({
        url: "/api/v1/accounts?name="+name+"&page="+page+"&size="+items,
        headers: { 'Authorization': GenHMAC() },
        success: function (data) {
            if (data.status !== "SUCCESS") {
                $("#findAccountRows").html("<tr><th scope=\"row\">&nbsp;</th><td colspan='6'>NO ACCOUNT WITH THAT CRITERIA IS FOUND</td></tr>");
            } else {
                let accountsListRecords = ""
                for (let i = 0; i < data.data.accounts.length; i++) {
                    accountsListRecords = accountsListRecords + "<tr><th scope=\"row\">"+ (i+1) +
                        "</th><td>" + data.data.accounts[i].account_number +
                        "</td><td>" + data.data.accounts[i].name +
                        "</td><td>" + data.data.accounts[i].coa +
                        "</td><td>" + data.data.accounts[i].alignment +
                        "</td><td>" + data.data.accounts[i].balance +
                        "</td><td><a href=\"#\" class=\"btn btn-info btn-circle btn-sm\" onclick='LoadAccountDetail(\""+data.data.accounts[i].account_number+"\");'><i class=\"fas fa-info-circle\"></i></a></td></tr>";
                }
                $("#findAccountRows").html(accountsListRecords);

                $("#btpGroupGoToAccountListPageTop").html("Page " + data.data.pagination.page);
                $("#btpGroupGoToAccountListPageBottom").html("Page " + data.data.pagination.page);

                let accountsListPages = "";
                for (let i = 1; i <= data.data.pagination.total_pages; i++) {
                    accountsListPages = accountsListPages + "<a class=\"dropdown-item\" href=\"#\" onclick='FindAccount(\""+name+"\","+i+","+items+");'>Page "+i+"</a>\n";
                }

                $("#btpGroupGoToAccountItemBottom").html(accountsListPages);
                $("#btpGroupGoToAccountItemTop").html(accountsListPages);

                $("#listAccountFirstPageTop").prop('disabled', data.data.pagination.is_first);
                $("#listAccountFirstPageTop").on("click", function() { FindAccount(name, data.data.pagination.first_page, items); });

                $("#listAccountPrevPageTop").prop('disabled', !data.data.pagination.have_previous);
                $("#listAccountPrevPageTop").on("click", function() { FindAccount(name, data.data.pagination.previous_page, items); });

                $("#listAccountNextPageTop").prop('disabled', !data.data.pagination.have_next);
                $("#listAccountNextPageTop").on("click", function() { FindAccount(name, data.data.pagination.next_page, items); });

                $("#listAccountLastPageTop").prop('disabled', data.data.pagination.is_last);
                $("#listAccountLastPageTop").on("click", function() { FindAccount(name, data.data.pagination.last_page, items); });

                $("#listAccountFirstPageBottom").prop('disabled', data.data.pagination.is_first);
                $("#listAccountFirstPageBottom").on("click", function() { FindAccount(name, data.data.pagination.first_page, items); });

                $("#listAccountPrevPageBottom").prop('disabled', !data.data.pagination.have_previous);
                $("#listAccountPrevPageBottom").on("click", function() { FindAccount(name, data.data.pagination.previous_page, items); });

                $("#listAccountNextPageBottom").prop('disabled', !data.data.pagination.have_next);
                $("#listAccountNextPageBottom").on("click", function() { FindAccount(name, data.data.pagination.next_page, items); });

                $("#listAccountLastPageBottom").prop('disabled', data.data.pagination.is_last);
                $("#listAccountLastPageBottom").on("click", function() { FindAccount(name, data.data.pagination.last_page, items); });
            }
        },
        error: function(data, errorThrown) {
            $("#findAccountRows").html("<tr><th scope=\"row\">&nbsp;</th><td colspan='6'>ERROR : "+errorThrown+"</td></tr>");
        },
        statusCode: {
            400: function (xhr) {
                window.alert("Seems you're input is wrong, check for date formats or mandatory fields.");
            },
            401: function (xhr) {
                window.alert("Seems your secret key is wrong.");
            }
        }
    });
}



function PopulateCurrencies() {
    $.ajax({
        url: "/api/v1/currencies",
        headers: { 'Authorization': GenHMAC() },
        success: function (data) {
            if (data.status !== "SUCCESS") {
                $("#findAccountRows").html("<tr><th scope=\"row\">&nbsp;</th><td colspan='5'>NO ACCOUNT WITH THAT CRITERIA IS FOUND</td></tr>");
            } else {
                let populatedCurrencies = "";
                for (let i = 0; i < data.data.length; i++) {
                    if (i === 0) {
                        populatedCurrencies = populatedCurrencies + "<option value=\""+data.data[i].code+"\" selected>"+data.data[i].code+" - "+data.data[i].name+"</option>";
                    } else {
                        populatedCurrencies = populatedCurrencies + "<option value=\""+data.data[i].code+"\">"+data.data[i].code+" - "+data.data[i].name+"</option>";
                    }
                }
                $("#newAccountCurrency").html(populatedCurrencies);
            }
        },
        error: function(data, errorThrown) {
            $("#findAccountRows").html("<tr><th scope=\"row\">&nbsp;</th><td colspan='5'>ERROR : "+errorThrown+"</td></tr>");
        },
        statusCode: {
            400: function (xhr) {
                window.alert("Seems you're input is wrong, check for date formats or mandatory fields.");
            },
            401: function (xhr) {
                window.alert("Seems your secret key is wrong.");
            }
        }
    });
}


function CreateNewCurrency() {
    let code = $("#newCurrencyCodeField").val();
    code = code.trim();

    let name = $("#newCurrencyNameField").val();
    name = name.trim();

    let exchange = $("#exchangeRateField").val();
    exchange = exchange.trim();

    if (code.length === 0 || name.length === 0 || exchange.length === 0) {
        $("#createCurrencyResult").html("<div class=\"card bg-danger text-white shadow\"><div class=\"card-body\">All fields is mandatory.</div></div>");
        return;
    }

    let exchangeNum = parseFloat(exchange);
    if (isNaN(exchangeNum)) {
        $("#createCurrencyResult").html("<div class=\"card bg-danger text-white shadow\"><div class=\"card-body\">Exchange rate must be number.</div></div>");
        return;
    }

    $.ajax({
        url: "/api/v1/currencies/" + code,
        headers: {'Authorization': GenHMAC() },
        type: "PUT",
        contentType: "application/json",
        data : JSON.stringify({
            name: name,
            exchange: exchangeNum,
            creator: "Dashboard"
        }),
        success: function (data) {
            if (data.status !== "SUCCESS") {
                $("#createCurrencyResult").html("<div class=\"card bg-danger text-white shadow\"><div class=\"card-body\">Failed create currency.\n<div class=\"text-white-50 small\">"+data.message+"</div></div></div>");
            } else {
                $("#createCurrencyResult").html("<div class=\"card bg-success text-white shadow\"><div class=\"card-body\">Currency successfully created</div></div>");
            }
        },
        error: function(data, errorThrown) {
            $("#createCurrencyResult").html("<div class=\"card bg-danger text-white shadow\"><div class=\"card-body\">Failed create currency.\n<div class=\"text-white-50 small\">Error : "+errorThrown+"</div></div></div>");
        },
        statusCode: {
            400: function (xhr) {
                window.alert("Seems you're input is wrong, check for date formats or mandatory fields.");
            },
            401: function (xhr) {
                window.alert("Seems your secret key is wrong.");
            }
        }
    });
}

function CreateNewAccount() {
    let accountNo = $("#newAccountNumberField").val();
    accountNo = accountNo.trim();

    let accountName = $("#newAccountNameField").val();
    accountName = accountName.trim();
    if (accountName.trim().length === 0) {
        $("#createAccountResult").html("<div class=\"card bg-danger text-white shadow\"><div class=\"card-body\">Account name must not be empty.</div></div>");
        return;
    }

    let description = $("#newAccountDescriptionField").val();
    let coa = $("#newAccountCOAField").val();
    let currency = $('#newAccountCurrency').find(":selected").val();
    let alignment = $('#newAccountAlignment').find(":selected").val();

    $.ajax({
        url: "/api/v1/accounts",
        headers: {'Authorization': GenHMAC() },
        type: "POST",
        contentType: "application/json",
        data : JSON.stringify({
            account_number: accountNo,
            name: accountName,
            description: description,
            coa: coa,
            currency: currency,
            alignment: alignment,
            creator: "Dashboard"
        }),
        success: function (data) {
            if (data.status !== "SUCCESS") {
                $("#createAccountResult").html("<div class=\"card bg-danger text-white shadow\"><div class=\"card-body\">Failed create account.\n<div class=\"text-white-50 small\">"+data.message+"</div></div></div>");
            } else {
                $("#createAccountResult").html("<div class=\"card bg-success text-white shadow\"><div class=\"card-body\">Account successfully created\n<div class=\"text-white-50 small\">Account number is : "+data.data+"</div></div></div>");
            }
        },
        error: function(data, errorThrown) {
            $("#createAccountResult").html("<div class=\"card bg-danger text-white shadow\"><div class=\"card-body\">Failed create account.\n<div class=\"text-white-50 small\">Error : "+errorThrown+"</div></div></div>");
        },
        statusCode: {
            400: function (xhr) {
                window.alert("Seems you're input is wrong, check for date formats or mandatory fields.");
            },
            401: function (xhr) {
                window.alert("Seems your secret key is wrong.");
            }
        }
    });
}


function PopulateCurrenciesForExchanges() {
    $.ajax({
        url: "/api/v1/currencies",
        headers: { 'Authorization': GenHMAC() },
        success: function (data) {
            if (data.status !== "SUCCESS") {
                console.error("error : " + data.message);
            } else {
                let populatedCurrencies = "";
                for (let i = 0; i < data.data.length; i++) {
                    if (i === 0) {
                        populatedCurrencies = populatedCurrencies + "<option value=\""+data.data[i].code+"\" selected>"+data.data[i].code+" - "+data.data[i].name+"</option>";
                    } else {
                        populatedCurrencies = populatedCurrencies + "<option value=\""+data.data[i].code+"\">"+data.data[i].code+" - "+data.data[i].name+"</option>";
                    }
                }
                $("#exchangeCurrencyTarget").html(populatedCurrencies);
                $("#exchangeCurrencySource").html(populatedCurrencies);
            }
        },
        error: function(data, errorThrown) {
            console.error("error : " + errorThrown);
        },
        statusCode: {
            400: function (xhr) {
                window.alert("Seems you're input is wrong, check for date formats or mandatory fields.");
            },
            401: function (xhr) {
                window.alert("Seems your secret key is wrong.");
            }
        }
    });
}


function CalculateExchange() {
    let target = $('#exchangeCurrencyTarget').find(":selected").val();
    let source = $('#exchangeCurrencySource').find(":selected").val();
    let samount = $("#exchangeCurrencyAmount").val();
    let amount = parseInt(samount);
    if (isNaN(amount)) {
        window.alert("Amount for exchange must be a number");
        return;
    }

    $.ajax({
        url: "/api/v1/exchange/" + source + "/" + target + "/" + samount.trim(),
        headers: {'Authorization': GenHMAC() },
        type: "GET",
        success: function (data) {
            if (data.status !== "SUCCESS") {
                window.alert("Error while calculating exchange : " + data.message);
            } else {
                $("#exchangeResult").text(samount + " " + source + " is equal to " + data.data + " " + target);
            }
        },
        error: function(data, errorThrown) {
            console.error(errorThrown)
        },
        statusCode: {
            400: function (xhr) {
                window.alert("Seems you're input is wrong, check for date formats or mandatory fields.");
            },
            401: function (xhr) {
                window.alert("Seems your secret key is wrong.");
            }
        }
    });
}

function PopulateAccountList() {
    let name = $("#searchNameOrAccountField").val();
    if (name.length < 3) {
        return;
    }
    $.ajax({
        url: "/api/v1/accounts?name="+name+"&page=1&size=10",
        headers: { 'Authorization': GenHMAC() },
        success: function (data) {
            if (data.status !== "SUCCESS") {
                console.error(data.message);
            } else {
                let accountsListOption = ""
                let accounts = data.data.accounts;
                for (let i = 0; i < accounts.length; i++) {
                    let acc = accounts[i];
                    if (i === 0) {
                        accountsListOption = accountsListOption + "<option value='" + acc.account_number + "' selected>" + acc.name + " - " + acc.currency + " - " + acc.alignment + "</option>";
                    } else {
                        accountsListOption = accountsListOption + "<option value='" + acc.account_number + "'>" + acc.name + " - " + acc.currency + " - " + acc.alignment + "</option>";
                    }
                }
                $("#accountSelectionList").html(accountsListOption);
            }
        },
        error: function(data, errorThrown) {
            console.error(errorThrown);
        },
        statusCode: {
            400: function (xhr) {
                window.alert("Seems you're input is wrong, check for date formats or mandatory fields.");
            },
            401: function (xhr) {
                window.alert("Seems your secret key is wrong.");
            }
        }
    });
}

let Transactions = [];

function PostJournal() {
    let desc = $("#newJournalDescription").val();
    let transacts = [];
    for (let i = 0; i < Transactions.length; i++) {
        transacts[i] = {
            account_number : Transactions[i].accNo,
            description : Transactions[i].trxDesc,
            alignment : Transactions[i].trxAlign,
            amount : Transactions[i].trxAmount
        }
    }
    let notif = $("#newJournalFormNotification");

    $.ajax({
        url: "/api/v1/journals",
        headers: {'Authorization': GenHMAC() },
        type: "POST",
        contentType: "application/json",
        data : JSON.stringify({
            description: desc,
            transactions: transacts,
            creator: "Dashboard"
        }),
        success: function (data) {
            if (data.status !== "SUCCESS") {
                notif.removeClass();
                notif.addClass("px-3 py-3 bg-gradient-danger text-white");
                notif.text("Failed create account : " + data.message);
            } else {
                notif.removeClass();
                notif.addClass("px-3 py-3 bg-gradient-success text-white");
                notif.text("Journal posted successfuly. Journal number is : " + data.data);
            }
        },
        error: function(data, errorThrown) {
            notif.removeClass();
            notif.addClass("px-3 py-3 bg-gradient-danger text-white");
            notif.text("Failed create account : " + errorThrown + ". Make sure journal is balanced, descriptions are filled.");
        },
        statusCode: {
            400: function (xhr) {
                notif.removeClass();
                notif.addClass("px-3 py-3 bg-gradient-danger text-white");
                notif.text("Failed create account. Make sure journal is balanced, descriptions are filled.");
            },
            401: function (xhr) {
                window.alert("Seems your secret key is wrong.")
            }
        }
    });
}

function RemoveTransaction(accNo) {
    let itoremove = -1;
    for (let i=0; i < Transactions.length; i++) {
        if (Transactions[i].accNo === accNo) {
            itoremove = i;
        }
    }
    if (itoremove >= 0) {
        Transactions.splice(itoremove,1);
        RenderTransactions(SortTransactions(Transactions));
    }
}

function AddTransactionToJournal() {
    let accNo = $('#accountSelectionList').find(":selected").val();
    let optText = $("#accountSelectionList").find(":selected").text();
    let arr = optText.split("-");
    let accName = arr[0].trim();
    let accCurrency = arr[1].trim();
    let accAlign = arr[2].trim();
    let trxAlign = $("input[name='alignmentOptionRadio']:checked").val();
    let trxDesc = $("#transactionDescriptionField").val();
    let samount = $("#transactionAmountField").val();
    let trxAmount = parseInt(samount);
    if (isNaN(trxAmount)) {
        window.alert("Transaction amount must be a number");
        return;
    }
    let toPush = {
        accNo : accNo,
        accName : accName,
        accAlign : accAlign,
        accCurrency : accCurrency,
        trxAlign : trxAlign,
        trxAmount : trxAmount,
        trxDesc : trxDesc
    };
    let found = false
    for (let i = 0; i < Transactions.length; i++) {
        if (Transactions[i].accNo === accNo) {
            Transactions[i] = toPush;
            found = true;
        }
    }
    if (!found) {
        Transactions.push(toPush);
    }
    RenderTransactions(SortTransactions(Transactions));
}

function ResetNewJournalForm() {
    Transactions.length = 0;
    RenderTransactions(Transactions);
}

function RenderTransactions(transacts) {
    let trxRows = "";

    let tdc = TotalDebitCredit(transacts);
    let notif = $("#newJournalFormNotification");
    if (transacts.length === 0) {
        notif.removeClass();
        notif.addClass("px-3 py-3 bg-gradient-info text-white");
        notif.text("Journal is empty. Create 2 or more account transactions using Account Lookup");
        $("#transactionAccountRows").html("<tr><td colspan='7'>Empty Journal</td></tr>");
    } else {
        for (let i = 0; i < transacts.length; i++) {
            let t = transacts[i];
            trxRows = trxRows + "<tr><th scope=\"row\">" + (i + 1) + "</th>" +
                "<td>" + t.accNo + "<br>" + t.accName + "</td>" +
                "<td>" + t.accCurrency + "</td>" +
                "<td>" + t.accAlign + "</td>" +
                "<td>" + t.trxDesc + "</td>";
            if (t.trxAlign === "DEBIT") {
                trxRows = trxRows + "<td>" + t.trxAmount + "</td><td></td>";
            } else {
                trxRows = trxRows + "<td></td><td>" + t.trxAmount + "</td>";
            }
            trxRows = trxRows + "<td>" +
                "<a href=\"#\" class=\"btn btn-info btn-circle btn-sm\">" +
                "<button type=\"button\" class=\"btn btn-danger btn-sm\" onClick=\"RemoveTransaction('" + t.accNo + "');\">Remove</button></a>" +
                "</td></tr>";
        }
        $("#transactionAccountRows").html(trxRows);
        $("#totalDebit").text(tdc.TotalDebit);
        $("#totalCredit").text(tdc.TotalCredit);
        if (tdc.TotalDebit !== tdc.TotalCredit) {
            notif.removeClass();
            notif.addClass("px-3 py-3 bg-gradient-warning text-white");
            notif.text("Journal is not yet balanced. You should adjust or add transactions  to make it balance");
        } else  {
            notif.removeClass();
            notif.addClass("px-3 py-3 bg-gradient-info text-white");
            notif.text("Journal is balanced. You can post or add transactions");
        }
    }
}

function TotalDebitCredit(transacts) {
    let td = 0;
    let tc = 0;
    for (let i = 0; i < transacts.length; i++) {
        if (transacts[i].trxAlign === "DEBIT") {
            td += transacts[i].trxAmount;
        } else {
            tc += transacts[i].trxAmount;
        }
    }
    return {
        TotalDebit : td,
        TotalCredit : tc
    }
}

function SortTransactions(transacts) {
    let ret = [];
    for (let i = 0; i < transacts.length; i++) {
        if (transacts[i].trxAlign === "DEBIT") {
            ret.push(transacts[i]);
        }
    }
    for (let i = 0; i < transacts.length; i++) {
        if (transacts[i].trxAlign === "CREDIT") {
            ret.push(transacts[i]);
        }
    }
    return ret;
}
