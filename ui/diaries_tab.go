package ui

import (
        "fmt"
        "image/color"

        "fyne.io/fyne/v2"
        "fyne.io/fyne/v2/canvas"
        "fyne.io/fyne/v2/container"
        "fyne.io/fyne/v2/dialog"
        "fyne.io/fyne/v2/theme"
        "fyne.io/fyne/v2/widget"

        "github.com/4codegit/edonish-auto/client"
)

// DiariesTab manages the Diaries (Дневник) tab with chat-like
// teacher/parent conversation, behavior comments, and signatures.
//
// In a real school diary:
//   - The teacher writes a comment about the student (praise, complaint, behavior note)
//   - The parent reads the comment, signs to acknowledge, and may write a response
//   - This creates a "conversation" between teacher and parent
type DiariesTab struct {
        controller Controller
        container  *fyne.Container

        // Filters
        classSel *widget.Select

        // State
        journalOpts   *client.JournalOptions
        selectedGroup *client.JournalGroup
        diaries       []client.DiaryEntry

        // UI
        diariesList *widget.List
        statusLabel *widget.Label
}

// NewDiariesTab creates a new DiariesTab.
func NewDiariesTab(c Controller) *DiariesTab {
        dt := &DiariesTab{
                controller:  c,
                statusLabel: widget.NewLabel("Выберите класс для загрузки дневников"),
        }
        dt.buildUI()
        go dt.loadJournalOptions()
        return dt
}

// Container returns the root container for this tab.
func (dt *DiariesTab) Container() fyne.CanvasObject {
        return dt.container
}

// buildUI creates the full UI layout for the diaries tab.
func (dt *DiariesTab) buildUI() {
        // Filter row
        dt.classSel = widget.NewSelect([]string{}, dt.onClassSelected)
        dt.classSel.PlaceHolder = "Выберите класс..."

        filterRow := container.NewHBox(
                widget.NewLabel("Класс:"),
                dt.classSel,
        )

        // Batch actions
        batchPraiseBtn := widget.NewButton("Подписать: Похвала", func() {
                dt.onBatchSignWithCategory(BehaviorPraise)
        })
        batchPraiseBtn.Importance = widget.HighImportance

        batchMixedBtn := widget.NewButton("Подписать: Смешанный", func() {
                dt.onBatchSignWithCategory(BehaviorMixed)
        })

        batchComplaintBtn := widget.NewButton("Подписать: Жалоба", func() {
                dt.onBatchSignWithCategory(BehaviorComplaint)
        })

        refreshBtn := widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
                if dt.selectedGroup != nil {
                        go dt.loadDiaries()
                }
        })

        actionRow := container.NewHBox(
                batchPraiseBtn,
                batchMixedBtn,
                batchComplaintBtn,
                refreshBtn,
        )

        // Placeholder
        placeholder := widget.NewLabelWithStyle(
                "Выберите класс для загрузки дневников\n\n"+
                        "В дневнике учитель пишет комментарий о поведении ученика,\n"+
                        "а родитель подписывает и может ответить.",
                fyne.TextAlignCenter,
                fyne.TextStyle{Italic: true},
        )

        dt.container = container.NewBorder(
                container.NewVBox(filterRow, actionRow, widget.NewSeparator()),
                dt.statusLabel,
                nil,
                nil,
                placeholder,
        )
}

// loadJournalOptions loads class list from API.
func (dt *DiariesTab) loadJournalOptions() {
        dt.statusLabel.SetText("Загрузка списка классов...")
        opts, err := dt.controller.GetClient().GetJournalOptions()
        if err != nil {
                fyne.Do(func() {
                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки настроек журнала: %v", err))
                })
                return
        }

        dt.journalOpts = opts

        classNames := make([]string, len(opts.Groups))
        for i, g := range opts.Groups {
                classNames[i] = fmt.Sprintf("%d %s", g.Number, g.Name)
        }

        fyne.Do(func() {
                dt.classSel.Options = classNames
                dt.classSel.Refresh()
                dt.statusLabel.SetText("Выберите класс")
                if len(classNames) > 0 {
                        dt.classSel.SetSelectedIndex(0)
                }
        })
}

// onClassSelected is called when a class is selected from the dropdown.
func (dt *DiariesTab) onClassSelected(selected string) {
        if dt.journalOpts == nil {
                return
        }

        var group *client.JournalGroup
        for i, g := range dt.journalOpts.Groups {
                gName := fmt.Sprintf("%d %s", g.Number, g.Name)
                if gName == selected {
                        group = &dt.journalOpts.Groups[i]
                        break
                }
        }

        if group == nil {
                return
        }

        dt.selectedGroup = group
        go dt.loadDiaries()
}

// loadDiaries calls GetDiaries API and rebuilds the list.
func (dt *DiariesTab) loadDiaries() {
        if dt.selectedGroup == nil {
                return
        }

        fyne.Do(func() {
                dt.statusLabel.SetText("Загрузка дневников...")
        })

        diaries, err := dt.controller.GetClient().GetDiaries(dt.selectedGroup.ID)

        fyne.Do(func() {
                if err != nil {
                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка загрузки дневников: %v", err))
                        return
                }

                dt.diaries = diaries
                dt.rebuildDiariesList()

                dt.statusLabel.SetText(fmt.Sprintf("Загружено дневников: %d", len(diaries)))
        })
}

// rebuildDiariesList builds the list showing each diary entry as a
// chat-like conversation between teacher and parent.
func (dt *DiariesTab) rebuildDiariesList() {
        if len(dt.diaries) == 0 {
                dt.container.Objects = []fyne.CanvasObject{
                        container.NewBorder(
                                container.NewVBox(
                                        container.NewHBox(widget.NewLabel("Класс:"), dt.classSel),
                                        container.NewHBox(
                                                widget.NewButton("Подписать: Похвала", func() { dt.onBatchSignWithCategory(BehaviorPraise) }),
                                                widget.NewButton("Подписать: Смешанный", func() { dt.onBatchSignWithCategory(BehaviorMixed) }),
                                                widget.NewButton("Подписать: Жалоба", func() { dt.onBatchSignWithCategory(BehaviorComplaint) }),
                                                widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
                                                        if dt.selectedGroup != nil {
                                                                go dt.loadDiaries()
                                                        }
                                                }),
                                        ),
                                        widget.NewSeparator(),
                                ),
                                dt.statusLabel,
                                nil,
                                nil,
                                widget.NewLabelWithStyle("Нет дневников для этого класса", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
                        ),
                }
                dt.container.Refresh()
                return
        }

        dt.diariesList = widget.NewList(
                func() int {
                        return len(dt.diaries)
                },
                func() fyne.CanvasObject {
                        // Header: subject + group
                        subjectLabel := widget.NewLabel("")
                        subjectLabel.TextStyle = fyne.TextStyle{Bold: true}
                        subjectLabel.Wrapping = fyne.TextWrapWord

                        // Teacher comment (chat bubble style)
                        teacherLabel := widget.NewLabel("")
                        teacherLabel.Wrapping = fyne.TextWrapWord

                        // Parent response area
                        parentLabel := widget.NewLabel("")
                        parentLabel.Wrapping = fyne.TextWrapWord

                        // Diligence + signatures row
                        diligenceLabel := widget.NewLabel("")
                        diligenceLabel.TextStyle = fyne.TextStyle{Bold: true}

                        sigsLabel := widget.NewLabel("")
                        sigsLabel.TextStyle = fyne.TextStyle{Italic: true}

                        return container.NewVBox(
                                subjectLabel,
                                teacherLabel,
                                parentLabel,
                                container.NewHBox(diligenceLabel, sigsLabel),
                        )
                },
                func(id widget.ListItemID, cell fyne.CanvasObject) {
                        if id < 0 || id >= len(dt.diaries) {
                                return
                        }
                        entry := dt.diaries[id]

                        vbox := cell.(*fyne.Container)

                        subjectLabel := vbox.Objects[0].(*widget.Label)
                        teacherLabel := vbox.Objects[1].(*widget.Label)
                        parentLabel := vbox.Objects[2].(*widget.Label)
                        bottomRow := vbox.Objects[3].(*fyne.Container)
                        diligenceLabel := bottomRow.Objects[0].(*widget.Label)
                        sigsLabel := bottomRow.Objects[1].(*widget.Label)

                        // Header
                        headerText := fmt.Sprintf("%s — %s", entry.SubjectName, entry.GroupName)
                        if entry.StudentLastName != "" || entry.StudentFirstName != "" {
                                headerText = fmt.Sprintf("%s %s: %s — %s",
                                        entry.StudentLastName, entry.StudentFirstName,
                                        entry.SubjectName, entry.GroupName)
                        }
                        subjectLabel.SetText(headerText)

                        // Teacher comment (chat bubble)
                        if entry.TeacherComment != "" {
                                teacherLabel.SetText("Учитель: " + entry.TeacherComment)
                                teacherLabel.TextStyle = fyne.TextStyle{}
                        } else if entry.DiligenceMark != "" {
                                // Use diligence mark description as a fallback comment
                                if desc, ok := DiligenceToBehaviorComment[entry.DiligenceMark]; ok {
                                        teacherLabel.SetText("Учитель: " + desc)
                                } else {
                                        teacherLabel.SetText("")
                                }
                        } else {
                                teacherLabel.SetText("Учитель: комментарий не оставлен")
                                teacherLabel.TextStyle = fyne.TextStyle{Italic: true}
                        }

                        // Parent response (chat bubble)
                        if entry.ParentComment != "" {
                                parentLabel.SetText("Родитель: " + entry.ParentComment)
                                parentLabel.TextStyle = fyne.TextStyle{}
                        } else if entry.ParentSigned {
                                parentLabel.SetText("Родитель: ознакомлен, подписано")
                                parentLabel.TextStyle = fyne.TextStyle{Italic: true}
                        } else {
                                parentLabel.SetText("")
                        }

                        // Diligence mark
                        if entry.DiligenceMark != "" {
                                diligenceLabel.SetText("Прилежание: " + entry.DiligenceMark)
                        } else {
                                diligenceLabel.SetText("Прилежание: —")
                        }

                        // Signature status
                        sigs := ""
                        if entry.ParentSigned {
                                sigs += "Род. подписано "
                        }
                        if entry.ManagerSigned {
                                sigs += "Рук. подписано"
                        }
                        if sigs == "" {
                                sigs = "Не подписано"
                        }
                        sigsLabel.SetText(sigs)
                },
        )

        dt.diariesList.OnSelected = func(id widget.ListItemID) {
                dt.diariesList.Unselect(id)
                dt.showDiaryDialog(id)
        }

        dt.container.Objects = []fyne.CanvasObject{
                container.NewBorder(
                        container.NewVBox(
                                container.NewHBox(widget.NewLabel("Класс:"), dt.classSel),
                                container.NewHBox(
                                        widget.NewButton("Подписать: Похвала", func() { dt.onBatchSignWithCategory(BehaviorPraise) }),
                                        widget.NewButton("Подписать: Смешанный", func() { dt.onBatchSignWithCategory(BehaviorMixed) }),
                                        widget.NewButton("Подписать: Жалоба", func() { dt.onBatchSignWithCategory(BehaviorComplaint) }),
                                        widget.NewButtonWithIcon("Обновить", theme.ViewRefreshIcon(), func() {
                                                if dt.selectedGroup != nil {
                                                        go dt.loadDiaries()
                                                }
                                        }),
                                ),
                                widget.NewSeparator(),
                        ),
                        dt.statusLabel,
                        nil,
                        nil,
                        dt.diariesList,
                ),
        }
        dt.container.Refresh()
}

// showDiaryDialog shows a dialog for a single diary entry with a chat-like
// interface for teacher comments and parent signatures.
func (dt *DiariesTab) showDiaryDialog(idx int) {
        if idx < 0 || idx >= len(dt.diaries) {
                return
        }

        entry := dt.diaries[idx]

        var dlg dialog.Dialog

        // --- HEADER ---
        headerText := fmt.Sprintf("%s — %s (%s)", entry.SubjectName, entry.GroupName, entry.QuarterName)
        if entry.StudentLastName != "" || entry.StudentFirstName != "" {
                headerText = fmt.Sprintf("%s %s: %s — %s (%s)",
                        entry.StudentLastName, entry.StudentFirstName,
                        entry.SubjectName, entry.GroupName, entry.QuarterName)
        }
        headerLabel := widget.NewLabelWithStyle(headerText, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

        // --- CONVERSATION AREA ---
        // Teacher's comment (like a chat bubble)
        teacherBubbleTitle := canvas.NewText("Учитель:", color.NRGBA{R: 37, G: 99, B: 235, A: 255})
        teacherBubbleTitle.TextStyle = fyne.TextStyle{Bold: true}
        teacherBubbleTitle.TextSize = 13

        var teacherCommentDisplay string
        if entry.TeacherComment != "" {
                teacherCommentDisplay = entry.TeacherComment
        } else if entry.DiligenceMark != "" {
                if desc, ok := DiligenceToBehaviorComment[entry.DiligenceMark]; ok {
                        teacherCommentDisplay = desc
                } else {
                        teacherCommentDisplay = "Прилежание: " + entry.DiligenceMark
                }
        } else {
                teacherCommentDisplay = "(нет комментария)"
        }
        teacherCommentLabel := widget.NewLabel(teacherCommentDisplay)
        teacherCommentLabel.Wrapping = fyne.TextWrapWord

        teacherBubble := container.NewVBox(
                teacherBubbleTitle,
                teacherCommentLabel,
        )

        // Parent's response (like a chat bubble)
        parentBubbleTitle := canvas.NewText("Родитель:", color.NRGBA{R: 22, G: 163, B: 74, A: 255})
        parentBubbleTitle.TextStyle = fyne.TextStyle{Bold: true}
        parentBubbleTitle.TextSize = 13

        var parentCommentDisplay string
        if entry.ParentComment != "" {
                parentCommentDisplay = entry.ParentComment
        } else if entry.ParentSigned {
                parentCommentDisplay = "Ознакомлен, подписано"
        } else {
                parentCommentDisplay = "(ожидает подписи)"
        }
        parentCommentLabel := widget.NewLabel(parentCommentDisplay)
        parentCommentLabel.Wrapping = fyne.TextWrapWord

        parentBubble := container.NewVBox(
                parentBubbleTitle,
                parentCommentLabel,
        )

        // Signature status
        parentStatus, parentColor := FormatSignedStatus(entry.ParentSigned)
        parentStatusText := canvas.NewText(fmt.Sprintf("Родители: %s", parentStatus), parentColor)
        parentStatusText.TextSize = 12

        managerStatus, managerColor := FormatSignedStatus(entry.ManagerSigned)
        managerStatusText := canvas.NewText(fmt.Sprintf("Руководитель: %s", managerStatus), managerColor)
        managerStatusText.TextSize = 12

        // --- ACTION AREA ---

        // Teacher: behavior category selector
        behaviorSel := widget.NewSelect(BehaviorCategories, nil)
        behaviorSel.PlaceHolder = "Категория комментария..."

        // Teacher: quick comment templates (changes when category is selected)
        quickCommentSel := widget.NewSelect([]string{}, nil)
        quickCommentSel.PlaceHolder = "Выберите шаблон комментария..."

        behaviorSel.OnChanged = func(cat string) {
                templates := BehaviorTemplates[BehaviorCategory(cat)]
                opts := make([]string, len(templates))
                copy(opts, templates)
                quickCommentSel.Options = opts
                quickCommentSel.Refresh()
                quickCommentSel.SetSelectedIndex(0)
        }

        // Teacher: custom comment entry
        commentEntry := widget.NewMultiLineEntry()
        commentEntry.SetPlaceHolder("Введите комментарий о поведении ученика...\nИли выберите шаблон выше")
        commentEntry.Wrapping = fyne.TextWrapWord

        // When a quick template is selected, fill the entry
        quickCommentSel.OnChanged = func(selected string) {
                if selected != "" {
                        commentEntry.SetText(selected)
                }
        }

        // Diligence selector (auto-set from behavior category, but can be overridden)
        diligenceSel := widget.NewSelect(DiligenceMarks, nil)
        diligenceSel.PlaceHolder = "Прилежание..."
        if entry.DiligenceMark != "" {
                diligenceSel.SetSelected(entry.DiligenceMark)
        }

        // Auto-set diligence when behavior category changes
        origBehaviorChanged := behaviorSel.OnChanged
        behaviorSel.OnChanged = func(cat string) {
                origBehaviorChanged(cat)
                if d, ok := BehaviorToDiligence[BehaviorCategory(cat)]; ok {
                        diligenceSel.SetSelected(d)
                }
        }

        // Teacher: write comment + set diligence button
        writeCommentBtn := widget.NewButton("Написать комментарий и прилежание", func() {
                comment := commentEntry.Text
                diligence := diligenceSel.Selected
                if comment == "" && diligence == "" {
                        dialog.ShowInformation("Внимание", "Напишите комментарий или выберите прилежание", dt.controller.GetWindow())
                        return
                }
                dlg.Hide()
                go dt.writeCommentAndDiligence(entry.DiaryID, diligence, comment, idx)
        })
        writeCommentBtn.Importance = widget.HighImportance

        // Parent: sign button
        parentBtn := widget.NewButton("Подпись родителей", func() {
                dlg.Hide()
                go dt.signDiary(entry.DiaryID, "parent", idx)
        })
        if entry.ParentSigned {
                parentBtn.Disable()
                parentBtn.SetText("Уже подписано (родители)")
        }

        // Manager: sign button
        managerBtn := widget.NewButton("Подпись руководителя", func() {
                dlg.Hide()
                go dt.signDiary(entry.DiaryID, "manager", idx)
        })
        if entry.ManagerSigned {
                managerBtn.Disable()
                managerBtn.SetText("Уже подписано (руководитель)")
        }

        // --- LAYOUT ---
        content := container.NewVBox(
                headerLabel,
                widget.NewSeparator(),
                // Conversation area
                widget.NewLabelWithStyle("Переписка:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                teacherBubble,
                parentBubble,
                widget.NewSeparator(),
                // Signature status
                parentStatusText,
                managerStatusText,
                widget.NewSeparator(),
                // Teacher action area
                widget.NewLabelWithStyle("Действия учителя:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                container.NewHBox(widget.NewLabel("Категория:"), behaviorSel),
                container.NewHBox(widget.NewLabel("Шаблон:"), quickCommentSel),
                commentEntry,
                container.NewHBox(widget.NewLabel("Прилежание:"), diligenceSel),
                writeCommentBtn,
                widget.NewSeparator(),
                // Signature action area
                widget.NewLabelWithStyle("Подписи:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
                container.NewHBox(parentBtn, managerBtn),
        )

        dialogTitle := fmt.Sprintf("Дневник: %s", entry.SubjectName)
        dlg = dialog.NewCustom(dialogTitle, "Закрыть", content, dt.controller.GetWindow())
        dlg.Show()
}

// writeCommentAndDiligence sends a teacher comment and diligence mark for a diary.
func (dt *DiariesTab) writeCommentAndDiligence(diaryID int, diligence string, comment string, idx int) {
        fyne.Do(func() {
                dt.statusLabel.SetText("Запись комментария...")
        })

        var err error
        apiClient := dt.controller.GetClient()

        if diligence != "" && comment != "" {
                // Set diligence with comment in one call
                err = apiClient.SetDiaryDiligenceWithComment(diaryID, diligence, comment)
        } else if diligence != "" {
                err = apiClient.SetDiaryDiligence(diaryID, diligence)
        } else if comment != "" {
                // Comment only — set with current or default diligence
                err = apiClient.SetDiaryDiligenceWithComment(diaryID, "Хорошо", comment)
        }

        fyne.Do(func() {
                if err != nil {
                        dialog.ShowError(fmt.Errorf("Ошибка записи: %v", err), dt.controller.GetWindow())
                        dt.statusLabel.SetText("Ошибка записи комментария")
                } else {
                        dt.statusLabel.SetText("Комментарий записан")
                        go dt.loadDiaries()
                }
        })
}

// signDiary signs a diary entry with the specified sign type.
func (dt *DiariesTab) signDiary(diaryID int, signType string, idx int) {
        fyne.Do(func() {
                dt.statusLabel.SetText(fmt.Sprintf("Подписание дневника (%s)...", signType))
        })

        err := dt.controller.GetClient().SignDiary(diaryID, signType)

        fyne.Do(func() {
                if err != nil {
                        dialog.ShowError(fmt.Errorf("Ошибка подписания дневника: %v", err), dt.controller.GetWindow())
                        dt.statusLabel.SetText("Ошибка подписания дневника")
                } else {
                        signLabel := "родителем"
                        if signType == "manager" {
                                signLabel = "руководителем"
                        }
                        dt.statusLabel.SetText(fmt.Sprintf("Дневник подписан %s", signLabel))
                        go dt.loadDiaries()
                }
        })
}

// onBatchSignWithCategory signs all unsigned diaries with a specific behavior category.
// This creates a comment from the category's template pool and sets the matching diligence.
func (dt *DiariesTab) onBatchSignWithCategory(category BehaviorCategory) {
        if len(dt.diaries) == 0 {
                dialog.ShowInformation("Внимание", "Нет дневников для подписания", dt.controller.GetWindow())
                return
        }

        // Filter unsigned diaries
        var unsigned []client.DiaryEntry
        for _, d := range dt.diaries {
                if !d.ParentSigned || !d.ManagerSigned || d.DiligenceMark == "" {
                        unsigned = append(unsigned, d)
                }
        }

        if len(unsigned) == 0 {
                dialog.ShowInformation("Готово", "Все дневники уже подписаны", dt.controller.GetWindow())
                return
        }

        diligence := BehaviorToDiligence[category]
        // Preview the first template as example
        templates := BehaviorTemplates[category]
        exampleComment := ""
        if len(templates) > 0 {
                exampleComment = templates[0]
        }

        confirmMsg := fmt.Sprintf(
                "Будет установлено:\n"+
                        "  Прилежание: «%s»\n"+
                        "  Категория комментария: «%s»\n"+
                        "  Пример комментария: «%s»\n\n"+
                        "Для %d дневников(я) будут записаны комментарии по порядку\n"+
                        "и подписаны все неподписанные записи.\n\n"+
                        "Продолжить?",
                diligence, string(category), exampleComment, len(unsigned),
        )

        dialog.ShowConfirm("Подписать все", confirmMsg, func(ok bool) {
                if !ok {
                        return
                }
                go dt.executeBatchSignWithComments(unsigned, category, diligence)
        }, dt.controller.GetWindow())
}

// executeBatchSignWithComments performs the batch signing operation with behavior comments.
func (dt *DiariesTab) executeBatchSignWithComments(unsigned []client.DiaryEntry, category BehaviorCategory, diligence string) {
        total := len(unsigned)
        apiClient := dt.controller.GetClient()

        for i, entry := range unsigned {
                progress := fmt.Sprintf("Обработка %d из %d: %s — %s", i+1, total, entry.GroupName, entry.SubjectName)
                fyne.Do(func() {
                        dt.statusLabel.SetText(progress)
                })

                // Set diligence with a sequential behavior comment
                comment := SequentialBehaviorComment(category, i)

                if entry.DiligenceMark == "" {
                        if err := apiClient.SetDiaryDiligenceWithComment(entry.DiaryID, diligence, comment); err != nil {
                                fyne.Do(func() {
                                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка установки прилежания (дневник %d): %v", entry.DiaryID, err))
                                })
                                continue
                        }
                } else if entry.TeacherComment == "" {
                        // Diligence already set, just add the comment
                        if err := apiClient.SetDiaryDiligenceWithComment(entry.DiaryID, entry.DiligenceMark, comment); err != nil {
                                fyne.Do(func() {
                                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка записи комментария (дневник %d): %v", entry.DiaryID, err))
                                })
                                continue
                        }
                }

                // Sign parent if not signed
                if !entry.ParentSigned {
                        if err := apiClient.SignDiary(entry.DiaryID, "parent"); err != nil {
                                fyne.Do(func() {
                                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка подписания родителем (дневник %d): %v", entry.DiaryID, err))
                                })
                                continue
                        }
                }

                // Sign manager if not signed
                if !entry.ManagerSigned {
                        if err := apiClient.SignDiary(entry.DiaryID, "manager"); err != nil {
                                fyne.Do(func() {
                                        dt.statusLabel.SetText(fmt.Sprintf("Ошибка подписания руководителем (дневник %d): %v", entry.DiaryID, err))
                                })
                                continue
                        }
                }
        }

        fyne.Do(func() {
                dt.statusLabel.SetText(fmt.Sprintf("Готово! Обработано %d дневников (прилежание: %s, категория: %s)", total, diligence, string(category)))
                go dt.loadDiaries()
        })
}

// Refresh updates the tab with new data from the dashboard context.
func (dt *DiariesTab) Refresh(students []client.Student, group *client.JournalGroup, subject *client.Subject, quarter *client.Quarter) {
        if group != nil && (dt.selectedGroup == nil || dt.selectedGroup.ID != group.ID) {
                dt.selectedGroup = group
                go dt.loadDiaries()
        }
}
